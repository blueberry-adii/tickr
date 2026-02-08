## Code Walkthrough

Here’s how the pieces fit together in **Tickr v2**, and how the system behaves in the real world — including failures, retries, and recovery.

---

### 1. The Entry Point (`cmd/server/main.go`)

This is where everything gets wired together. I’m using **dependency injection** to keep things decoupled and testable.

- **Graceful Shutdown**  
  This is non-negotiable. I use `signal.Notify` to catch `SIGINT / SIGTERM`. When that happens, I cancel the global `context`.

  That cancellation:

  - Stops the scheduler loop
  - Stops Redis consumers
  - Signals all workers to stop accepting new jobs
  - Allows in-flight jobs to finish execution

- **WaitGroups**  
  Every long-running goroutine (scheduler + workers) is tracked using a `sync.WaitGroup`.  
  The main process does **not exit** until:

  - all workers finish their current job
  - the scheduler shuts down cleanly

- **Worker Pool**  
  A fixed pool of workers (currently 5) is spawned at startup. Workers are long-lived and block on channels instead of polling.

---

### 2. The Scheduler (`internal/scheduler`)

This is the **control plane** of Tickr. It owns time, orchestration, and recovery.

- **Durable Source of Truth**  
  MySQL is the source of truth for all jobs and their state. Redis is treated as a **disposable scheduling index**, not trusted state.

- **Immediate Jobs**  
  Jobs ready for execution live in a Redis List (`tickr:queue:ready`) and are consumed using a single blocking `BRPOP`.

- **Delayed Jobs (Waiting Queue)**  
  Delayed jobs live in a Redis Sorted Set (`tickr:queue:waiting`) with `executeAt` as the score.

- **Event-Driven Scheduling (No Polling Hot Path)**  
  Instead of polling every second, the scheduler:

  - Computes the next execution time (`nextExecutionTime`)
  - Sleeps exactly until that moment using a timer
  - Wakes up immediately if a new earlier job arrives via a notification channel (`wqCh`)

- **Redis Failure Handling**  
  Redis is assumed to fail.

  - If Redis goes down, the scheduler blocks safely and waits for reconnection
  - Once Redis is back, the scheduler re-evaluates time and flushes overdue jobs
  - If Redis state is lost, the scheduler **rebuilds Redis from MySQL**

- **Time Discontinuity Handling**  
  Jobs whose scheduled time passed while Redis or the scheduler was down are detected and executed immediately after recovery.

---

### 3. Redis Fetcher (`PopReadyQueue`)

This runs as a **single goroutine**, separate from workers.

```go
BRPOP tickr:queue:ready
```

- Blocks until a job is available
- Unmarshals the Redis payload
- Pushes the job into the internal JobCh
- Handles Redis disconnects gracefully:
- waits for Redis to come back
- triggers recovery if state was lost

This keeps Redis consumption centralized and avoids multiple workers competing over Redis.

---

### 4. The Workers (internal/worker)

Workers are pure executors. They don’t know about Redis, scheduling, or time.

- Worker Loop
  - Blocks on JobCh
  - Exits immediately when the global context is cancelled
- Execution Flow
  1. Fetch full job details from MySQL using the JobID
  2. Mark job as executing, set worker_id, set started_at
  3. Execute the job via the Executor
  4. Increment attempt count
  5. Persist final state:
  - completed on success
  - retrying with delayed requeue
  - failed when max attempts are reached
- Retries
  - Retries are bounded by maxAttempts
  - Retry delay increases per attempt
  - Workers never sleep
    — they compute executeAt and hand control back to the scheduler

---

### 5. The Executor (internal/worker/executor.go)

This is a simple routing layer.

- Uses a switch on jobType (e.g. email, report)
- Handles job-specific JSON unmarshalling
- Keeps workers generic and stateless

---

### 6. Concurrency Model

- Pipeline Pattern
- Redis Fetcher → JobCh → Workers
- Go Channels
- Provide backpressure naturally
- Workers block when no jobs are available
- Scheduler blocks when workers are busy

No busy loops. No hot polling. No wasted CPU.

---

### Design Decisions

1. **Why Redis + MySQL?**
   MySQL is the durable source of truth. Redis is fast, atomic, and disposable. Losing Redis state is acceptable; losing MySQL state is not.

2. **Why Event-Driven Scheduling?**
   Polling wastes CPU and hides timing bugs. The scheduler wakes up exactly when needed and reacts immediately to new jobs.

3. **Why Single Redis Consumer?**
   Centralizing Redis consumption avoids race conditions and simplifies recovery logic.

4. **Why Workers Don’t Sleep?**
   Workers compute when a retry should happen, but never wait. Time belongs to the scheduler.

5. **Graceful Shutdown**
   The system guarantees zero job loss. On shutdown, in-flight jobs complete and state is persisted before exit.

---

Tickr v2 is designed to be correct under failure
