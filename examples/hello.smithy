
namespace examples

service hello {
    version: "0.0",
    operations: [Hello]
}

///
/// A minimal hello world action
///
@http(method: "GET", uri: "/hello", code: 200)
@readonly
operation Hello {
    input: HelloInput,
    output: HelloOutput,
}

structure HelloOutput {
  @httpPayload
  greeting: String,
}

structure HelloInput {
  @httpQuery("caller")
  caller: String,
}


apply Hello @examples([
  {
    "title": "helloExample",
    "documentation": "An example of the Hello operation",
    "input": {
      "caller": "Lee"
    },
    "output": {
      "greeting": "Hello, Lee"
    }
  }
])
