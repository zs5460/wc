# wc

Minimalist, dependency-free, dedicated for sending text messages via WeCom (Enterprise WeChat).

## Quick Start

```go
package main

import (
    "github.com/zs5460/wc"
)

func main() {
    app, _ := wc.New("yourAppID","yourSecret","yourAgentId")

    app.Send("@all", "Hello,World!")
}
```

## License

MIT
