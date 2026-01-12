# API Endpoints

As of now there are only two API endpoints for this application

### **GET** /api/v1/health/

This endpoint is a health endpoint to make sure whether the server is running.
If it returns http error then the server has crashed or isn't running

Example:

```bash
curl localhost:8080/api/v1/health/

-> {"status":200,"message":"REST API Up and Working!!!","data":null,"success":true}
```

### **POST** /api/v1/jobs

This endpoint is the essential of this application. You send a post request on this endpoint with the following fields in the request body

1. jobtype: Right now there are only two jobtypes the workers can recognize `"email | report"`

2. payload: Data the worker needs to execute the job.

   - Payload for email:
     `{"to":"example@email.com", "from": "---", "body": "Email Body"}`

   - Payload for report: `{"title":"report title", "body":"report body", "time":10}`
     - time field requires the time in seconds, you want to publish the report after

3. delay: Seconds you want to delay the task/schedule the job

## Examples:

```bash
curl -X POST localhost:8080/api/v1/jobs/ \
-H "Content-Type: application/json" \
-d '{"jobType":"email", "payload":{"to":"john@gmail.com", "from":"aditya@proton.me", "body":"Email Messaging is working"}, "delay":5}'

-> {"status":200,"message":"Job Submitted!!!","data":null,"success":true}
```

---

```bash
curl -X POST localhost:8080/api/v1/jobs/ \
-H "Content-Type: application/json" \
-d '{"jobType":"report", "payload": {"title":"Weather Report", "body":"Weather today is Rainy", "time":10}, "delay":5}'

-> {"status":200,"message":"Job Submitted!!!","data":null,"success":true}
```

## Server Logs:

```bash
2026/01/12 08:38:21 Scheduler Idle
2026/01/12 08:38:21 Listening on Port 8080
2026/01/12 08:38:21 worker 2 idle
2026/01/12 08:38:21 worker 3 idle
2026/01/12 08:38:21 worker 4 idle
2026/01/12 08:38:21 worker 5 idle
2026/01/12 08:38:21 worker 1 idle
2026/01/12 08:43:38 POST /api/v1/jobs/ 2.050194ms
2026/01/12 08:43:38 New Job in WQ
2026/01/12 08:43:38 Scheduler Idle
2026/01/12 08:43:43 moving job eb259757-3c45-4213-9d5f-71e851c89f61 from waiting to ready queue
2026/01/12 08:43:43 Scheduler Idle
2026/01/12 08:43:43 worker 2 took job eb259757-3c45-4213-9d5f-71e851c89f61
2026/01/12 08:43:43 Sending email from aditya@proton.me to john@gmail.com
2026/01/12 08:43:48 Sent Email: Email Messaging is working
2026/01/12 08:43:48 worker 2 idle
2026/01/12 08:49:13 GET /api/v1/health/ 30.097µs
2026/01/12 09:04:52 POST /api/v1/jobs/ 1.869634ms
2026/01/12 09:04:52 New Job in WQ
2026/01/12 09:04:52 Scheduler Idle
2026/01/12 09:04:57 moving job 612de15c-6b10-441c-9e1b-5bf8c4698b39 from waiting to ready queue
2026/01/12 09:04:57 Scheduler Idle
2026/01/12 09:04:57 worker 3 took job 612de15c-6b10-441c-9e1b-5bf8c4698b39
2026/01/12 09:04:57 Scheduled report for 10 seconds
2026/01/12 09:05:07 Title: Weather Report | Body: Weather today is Rainy
2026/01/12 09:05:07 worker 3 idle
```

---
