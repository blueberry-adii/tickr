## Code Walkthrough

Here is how the pieces fit together across the codebase:

### 1. The Entry Point (`cmd/server/main.go`)

This is where everything wires up. I'm using **Dependency Injection** here to keep things testable.

- **Graceful Shutdown**: This is crucial. I use `signal.Notify` to catch `CTRL+C`. When that happens, I cancel the global `context`.
  The system captures `SIGINT/SIGTERM`.
  - Stops the scheduler.
  - Closes channels.
  - Waits for all workers to finish their active tasks (`sync.WaitGroup`) before exiting.
- **WaitGroups**: I track every single active goroutine (workers and scheduler). I don't let the main process exit until every worker has finished its current job (`wg.Wait()`).
- **Worker Pool**: I spin up a fixed number of workers (currently 5) to handle jobs in parallel.

### 2. The Scheduler (`internal/scheduler`)

This is the heart of the queueing logic.

- **Immediate Jobs**: Go straight to a Redis List (`tickr:queue:ready`).
- **Delayed Jobs** (`scheduler.go`): these go to a Redis Sorted Set (`tickr:queue:waiting`) with the timestamp as the score.
- **Intelligent Polling**: Instead of a dumb 1 second loop, the scheduler uses `nextExecutionTime` to sleep exactly until the next job is due.
- **Real-Time Wakeup**: Implemented a notification channel (`wqCh`) so that if a new delayed job is added, the scheduler wakes up immediately to recalculate its sleep time.

### 3. The Workers (`internal/worker`)

- **The Loop**: Each worker runs an infinite loop. They use scheduler job channel to block the loop and wait until either context is cancelled or they receive a job.
- **The Executor**: Once a worker gets a job, it hands it off to the `Executor`.
- **Executor Pattern** (`executor.go`): This is a switch statement that routes the job to the right function based on its type (`email`, `report`). It handles the JSON unmarshalling for the specific payload.

### 4. High-Concurrency Worker Pool

- **Pipeline Pattern**: Separated the "Fetcher" (`PopReadyQueue` goroutine) from the "Processors" (Workers).
- **Go Channels**: Jobs flow from Redis -> Fetcher -> `JobCh` -> Workers. This enables idiomatic Go concurrency using `select`.

### 5. The API (`internal/api`)

- **Endpoints**:

  - `POST /api/v1/jobs`: Accepts a JSON payload. It generates a UUID for the job and pushes it to the scheduler.
  - `GET /api/v1/health`: Simple health check.

## Design Decisions

1.  **Why Redis?**: It's fast and atomic. `BRPop` gives us reliable queue semantics without complex locking.
2.  **Why UUIDs in API?**: The frontend shouldn't trust the client for IDs. We generate them on the server to ensure uniqueness.
3.  **Why Sorted Sets for Delays?**: It's the standard pattern for delayed queues. We can efficiently query "give me everything with score < now".
4.  **Graceful Shutdown**: I wanted to ensure zero data loss. If you deploy a new version, the old one finishes its active jobs before dying.
