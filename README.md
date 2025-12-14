# wc


极简，无依赖，专用于企业微信发送文本消息。

## 快速开始


```go
package main

import (
	"github.com/zs5460/wc"
)

func main() {
	
	app, _ := wc.New("yourAppID","yourSecret","yourAgentId")

    // 推送消息
	app.Send("@all", "Hello,World!")
}

```

## License

MIT
