# Welcome to protoxy 👋
![Version](https://img.shields.io/badge/version-0.1-blue.svg?cacheSeconds=2592000)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](#)

> A proxy server than converts JSON request bodies to protocol buffers.

## Install

```sh
go get github.com/camgraff/protoxy
```

## Usage

1. Start the server by specifying the path to your proto file and optional port.
```sh
protoxy -p ./protos/example.proto --port 7777
```

2. Configure Postman to send request through the Proxy server.

3. Add your fully-qualified message names as params in the Content-Type header. For example, if I have CreatePost and PostResponse messages defined in an `example` proto package
```
Content-Type: application/x-protobuf; reqMsg=example.CreatePost; respMsg=example.PostResponse
```

## Author

👤 **Cam Graff**

* Github: [@camgraff](https://github.com/camgraff)
* LinkedIn: [@camgraff](https://linkedin.com/in/camgraff)

## 🤝 Contributing

Contributions, issues and feature requests are welcome!

Feel free to check [issues page](https://github.com/camgraff/protoxy/issues). 

## Show your support

Give a ⭐️ if this project helped you!


***
