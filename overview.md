# Tickr - Project Overview

Hey! This is a rundown of **Tickr**, a robust asynchronous job processing system I built using Go and Redis. It's designed to offload heavy tasks from APIs so they don't block, handling everything in the background with concurrency and reliability.

## The Core Concept

The idea is simple:
1.  **API**: Accepts a job (like "send email" or "generate report").
2.  **Scheduler**: Puts it in a queue. If it's a delayed job, it holds it until it's ready.
3.  **Workers**: A pool of background workers pick up jobs and execute them.

## Architecture & Tech Stack

*   **Language**: Go (Golang) - chosen for its amazing concurrency (`goroutines` and `channels`).
*   **Storage**: Redis - used as the queue engine.
    *   **Lists (`LPUSH`/`BRPOP`)**: For the "Ready" queue (the hot path).
    *   **Sorted Sets (`ZADD`/`ZRANGE`)**: For the "Waiting" queue (delayed jobs).
*   **Architecture**: Fan-out pattern. One API producers, multiple Worker consumers.

## Code Walkthrough

Here is how the pieces fit together across the codebase:

### 1. The Entry Point (`cmd/server/main.go`)
This is where everything wires up. I'm using **Dependency Injection** here to keep things testable.
*   **Graceful Shutdown**: This is crucial. I use `signal.Notify` to catch `CTRL+C`. When that happens, I cancel a global `context`.
*   **WaitGroups**: I track every single active goroutine (workers and scheduler). I don't let the main process exit until every worker has finished its current job (`wg.Wait()`).
*   **Worker Pool**: I spin up a fixed number of workers (currently 5) to handle jobs in parallel.

### 2. The Scheduler (`internal/scheduler`)
This is the heart of the queueing logic.
*   **Immediate Jobs**: Go straight to a Redis List (`tickr:queue:ready`).
*   **Delayed Jobs** (`scheduler.go`): these go to a Redis Sorted Set (`tickr:queue:waiting`) with the timestamp as the score.
*   **The Poller**: I have a background loop (`Run`) that ticks every second. It checks the Sorted Set for jobs whose time has come, pops them, and moves them to the Ready Queue.

### 3. The Workers (`internal/worker`)
*   **The Loop**: Each worker runs an infinite loop. They use `BRPop` (Blocking Pop) to fetch jobs from Redis. This is efficient—it "sleeps" until a job arrives, rather than busy-waiting.
*   **The Executor**: Once a worker gets a job, it hands it off to the `Executor`.
*   **Executor Pattern** (`executor.go`): This is a switch statement that routes the job to the right function based on its type (`email`, `report`). It handles the JSON unmarshalling for the specific payload.

### 4. The API (`internal/api`)
*   **Endpoints**:
    *   `POST /api/v1/jobs`: Accepts a JSON payload. It generates a UUID for the job and pushes it to the scheduler.
    *   `GET /api/v1/health`: Simple health check.

## Design Decisions / "Why I did it this way"

1.  **Why Redis?**: It's fast and atomic. `BRPop` gives us reliable queue semantics without complex locking.
2.  **Why UUIDs in API?**: The frontend shouldn't trust the client for IDs. We generate them on the server to ensure uniqueness.
3.  **Why Sorted Sets for Delays?**: It's the standard pattern for delayed queues. We can efficiently query "give me everything with score < now".
4.  **Graceful Shutdown**: I wanted to ensure zero data loss. If you deploy a new version, the old one finishes its active jobs before dying.

## Future Improvements

If I were to take this to v2, I'd probably add:
*   **Dead Letter Queues**: For jobs that fail repeatedly.
*   **Retry Logic**: Currently if a job fails, we just log it. We should probably re-enqueue it with an exponential backoff.
*   **Dashboard**: A nice UI to see how many jobs are pending.
