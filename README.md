# Tickr v2 - (Concurrent Job Scheduler)

**Tickr** is a robust asynchronous job processing system I built using Go, Redis and MySQL. It's designed to offload heavy tasks from APIs so they don't block, handling everything in the background with concurrency and reliability.

#### Refer [SETUP](./docs/setup.md) to setup and run this project locally

## The Core Concept

The idea is simple:

1.  **API**: Accepts a job (like "send email" or "generate report").
2.  **Scheduler**: Puts it in a queue. If it's a delayed job, it holds it until it's ready.
3.  **Workers**: A pool of background workers pick up jobs from ready queue and execute them.

## Architecture & Tech Stack

![Architecture Diagram](./assets/diagram.svg)

- **Language**: Go (Golang) - chosen for its amazing concurrency (`goroutines` and `channels`).
- **Storage (Source of Truth)**: MySQL - used to store job details and states as the source of truth.
- **In Memory Storage**: Redis - used as the queue engine.
  - **Lists (`LPUSH`/`BRPOP`)**: For the Ready queue.
  - **Sorted Sets (`ZADD`/`ZRANGE`)**: For the Waiting queue (delayed jobs).
- **Architecture**: Fan-out pattern. One API producers, multiple Worker consumers.

More on <--- [ARCHITECTURE](./docs/architecture.md) --->

## Future Improvements

- **Dead Letter Queues**: For jobs that fail repeatedly.
- **Metrics**: A nice UI or Analytics to see how many jobs are pending.
- **SDKs**: Create SDKs for JS and Python
- **Exponential Backoff** - Right Retry Logic implements Linear Backoff
- **HA Scheduler** - 2 or 3 Schedulers monitoring each other, if one fails, other takes over

## Feature Checklist

| Feature                         | Status | Implementation Details                                                       |
| :------------------------------ | :----- | :--------------------------------------------------------------------------- |
| **Job Submission API**          | ✅     | `POST /jobs` persists job in MySQL (source of truth), returns JobID.         |
| **Instant Jobs**                | ✅     | Redis `LPUSH` -> single scheduler `BRPOP` -> worker pool                     |
| **Delayed Jobs (WQ)**           | ✅     | Redis `ZADD` with executeAt. Scheduler computes next wake-up dynamically.    |
| **Event-Driven Scheduler**      | ✅     | No polling hot path; timer + channels + Redis blocking ops.                  |
| **Execution Logic**             | ✅     | Workers execute jobs, update state atomically in MySQL.                      |
| **Retry System**                | ✅     | Bounded retries with delay; attempt-based backoff; retry survives Redis loss |
| **Failure Handling**            | ✅     | Redis downtime + state loss fully recoverable from MySQL.                    |
| **Time Discontinuity** Handling | ✅     | Overdue jobs execute immediately after recovery.                             |
| **Graceful Shutdown**           | ✅     | In-flight jobs complete; state persisted safely.                             |
| **Logging**                     | ✅     | Detailed lifecycle logs (scheduler, worker, retries, recovery).              |
| **Scalability**                 | ✅     | Configurable worker pool; workers never block/sleep.                         |
| **Durable State**               | ✅     | MySQL as source of truth; Redis treated as disposable index.                 |

### --- Made By Aditya Prasad ---

[**LinkedIn**](https://www.linkedin.com/in/aditya-prasad-095ab9329/) | [**Dev.to**](https://dev.to/blueberry_adii) | [**X.com**](https://x.com/AdityaPrasad455)
