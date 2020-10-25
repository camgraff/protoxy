# Welcome to Protoxy 👋
[![codecov](https://codecov.io/gh/camgraff/protoxy/branch/master/graph/badge.svg)](https://codecov.io/gh/camgraff/protoxy)
[![Go Report Card](https://goreportcard.com/badge/github.com/camgraff/protoxy)](https://goreportcard.com/report/github.com/camgraff/protoxy)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](#)

## What is Protoxy?
Protoxy allows you to test your REST APIs that use [Protocol Buffer](https://developers.google.com/protocol-buffers) serialization through Postman and other API testing tools which do not natively support Protobuf encoding. Protoxy is a proxy server that converts the JSON in your request body to the appropriate Protobuf message type and transforms the response from your back-end back into JSON. You don't need to make any changes to your source code to use Protoxy.

## Install

```
go get github.com/camgraff/protoxy
```

## Usage
Consider a proto file located at `./protos/example.proto` that looks like this:

```
syntax = "proto3";
package example;

message ExampleRequest {
    string text = 1;
    int32 number = 2;
    repeated string list = 3;
}

message ExampleResponse {
    string text = 1;
}

```

1. Start the server by specifying your import paths, proto file names, and optional port.

    ```
    protoxy -I ./protos/ --port 7777 example.proto
    ```

2. Configure Postman to send requests through the Proxy server.
    ![Postman proxy config](https://raw.githubusercontent.com/camgraff/protoxy/master/media/postman-config.png)

3. Add your fully-qualified message names as params in the Content-Type header.

    ```
    Content-Type: application/x-protobuf; reqMsg="example.ExampleRequest"; respMsg="example.ExampleResponse";
    ```

4. Send your request as a raw JSON body.

    ```
    {
      "text": "some text",
      "number": 123,
      "list": ["this", "is", "a", "list"]
    }
    ```

    The response is:

    ```
    {
      "text": "this response was automagically converted to JSON"
    }
    ```

### Using Protobuf in Query String

Protoxy also supports sending protobuf messages as a base64 encoded query string in the URL. To do this, add an additional param `qs` in the header whose value corresponds to the query string parameter. For example:

```
Content-Type: application/x-protobuf; reqMsg="example.ExampleRequest"; respMsg="example.ExampleResponse"; qs="proto_body";
```

This will result in a URL like:

```
http://example.com?proto_body={base64 encoding of example.ExampleRequest}
```

### Handling Multiple Response Message Types
If your API sends multiple response message types, the `respMsg` parameter accepts a comma-seperated list of values.

```
Content-Type: application/x-protobuf; reqMsg="example.ExampleRequest"; respMsg="example.ExampleResponse,example.DifferentResponse";
```

Note: Protoxy will attempt to unmarshal your proto messages into each type of response and will send the first successful one. This can produce unexpected results because the same wire-format message can successfully be unmarshalled into multiple proto message types depending on the fields in the proto message. If possible, it is best to ensure that you back-end server returns only one response type per route.


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
