# f2p

A simple peer-to-peer file sharing tool that enables file transfers through a relay server.

## Overview

`f2p` is a command-line utility that allows users to share files between computers using a relay server. The tool supports three main operations:

- Starting a relay server
- Sending files to the relay server
- Receiving files from the relay server using a unique ID

## Features

- Simple command-line interface
- Direct file transfer via HTTP
- Unique ID generation for each file transfer
- Efficient streaming of large files (no memory loading)
- Configurable server port and host

## Installation

First, make sure you have Go installed on your system. Then you can install `f2p`:

```bash
go mod tidy
go build -o f2p .
```

## Usage

### Starting a Server

To start the relay server:

```bash
./f2p server
```

The server will start on port 9000 by default. You can change the port using the `P2PSHARE_PORT` environment variable:

```bash
P2PSHARE_PORT=8080 ./f2p server
```

Additionally, you can specify the server host using the `P2PSHARE_HOST` environment variable:

```bash
P2PSHARE_HOST=http://yourdomain.com:8080 ./f2p server
```

### Sending a File

To send a file through the relay server:

```bash
./f2p send <url> <filepath>
```

For example:

```bash
./f2p send http://localhost:9000 /path/to/your/file.txt
```

This will upload the file to the server, which will provide a unique ID and download link.

## How It Works

1. The relay server is started and listens for incoming connections
2. When a file is sent, it's streamed to the server without loading into memory, which assigns it a unique ID
3. The server provides a download link that includes the unique ID
4. A receiver can use the ID to retrieve the file from the server

## Security Considerations

- Files are streamed directly without loading into memory during transfer
- No authentication is required to send or receive files
- Use with caution on untrusted networks

## Environment Variables

- `P2PSHARE_PORT`: Port number for the server (default: 9000)
- `P2PSHARE_HOST`: Host address for the server (default: <http://localhost:9000>)

## Memory Usage

The application handles large files efficiently by streaming them directly through the HTTP request/response cycle without loading the entire file into memory. This allows for the transfer of files of any size without memory constraints.

## Building from Source

```bash
git clone <repository-url>
cd f2p
go build -o f2p .
```

## License

This project is open source and available under the MIT License.

