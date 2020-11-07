namespace example

service multi {
    version: "0.0",
    operations: [GetFoo, GetBar, GetGlorp]
}

@http(method: "GET", uri: "/glorp/{caller}", code: 200)
@readonly
operation GetGlorp {
    input: GetGlorpInput,
    output: GetGlorpOutput,
}

structure GetGlorpOutput {
    @httpPayload
    greeting: Greeting,
}

@http(method: "GET", uri: "/foo/{caller}", code: 200)
@tags(["FooService"])
@readonly
operation GetFoo {
    input: GetFooInput,
    output: GetFooOutput,
}

structure GetFooOutput {
    @httpPayload
    greeting: Greeting,
}

@http(method: "GET", uri: "/bar/{caller}", code: 200)
@tags(["BarService"])
@readonly
operation GetBar {
    input: GetBarInput,
    output: GetBarOutput,
}

structure GetBarOutput {
    @httpPayload
    greeting: Greeting,
}

structure GetGlorpInput {
    @httpLabel
    @required
    caller: String,
}


structure GetFooInput {
    @httpLabel
    @required
    caller: String,
}

structure GetBarInput {
    @httpLabel
    @required
    caller: String,
}

structure Greeting {
    message: String,
}

