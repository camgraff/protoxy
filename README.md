# Welcome to Protoxy 👋
![Version](https://img.shields.io/badge/version-0.1-blue.svg?cacheSeconds=2592000)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](#)

## What is Protoxy?
Protoxy allows you to test your REST APIs that use [Protocol Buffer](https://developers.google.com/protocol-buffers) serialization through Postman and other API testing tools which do not natively support Protobuf encoding. Protoxy spins up a reverse proxy server that converts the JSON in your request body to the appropriate Protobuf message type. You don't need to make any changes to your source code to use Protoxy.

## Install

```sh
go get github.com/camgraff/protoxy
```

## Usage

1. Start the server by specifying your import paths, proto file names, and optional port
```sh
protoxy -I ./protos/ -p example.proto --port 7777
```

2. Configure Postman to send requests through the Proxy server.
![Postman proxy config](https://raw.githubusercontent.com/camgraff/protoxy/master/media/postman-config.png)

3. Add your fully-qualified message names as params in the Content-Type header. For example, if I have CreatePost and PostResponse messages defined in an `example` proto package:
```
Content-Type: application/x-protobuf; reqMsg=example.CreatePost; respMsg=example.PostResponse
```
Protoxy also supports sending protobuf messages as a base64 encoded querystring in the URL. To do this, add an additional param in the header like so:
```
Content-Type: application/x-protobuf; reqMsg=example.CreatePost; respMsg=example.PostResponse; qs=proto_body
```
This will result in a URL like:
```
http://example.com?proto_body={base64 encoding of example.CreatePost}
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
