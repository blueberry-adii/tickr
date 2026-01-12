# Tickr Setup

To run this project locally, open your terminal and enter

```bash
git clone github.com/blueberry-adii/tickr.git
```

Change directory into project root, and open your code editor

```bash
cd tickr
code .
```

Make sure you have Redis running on **Port: 6379**, if not then start redis using docker

```bash
docker run -d --name redis -p 6379:6379 redis
```

Compile the source and run it

```bash
go build ./cmd/server/main.go
./main
```

After the server starts and scheduler & workers are active, start sending http requests to the API endpoints

### More on that here -> [API Docs](./api.md)
