package tidb

import (
	"fmt"
	"testing"
)

func TestParseConfig(t *testing.T) {
	fmt.Println(parseConfig("/home/ofey/Downloads/tikv-test-config.toml"))
	fmt.Println(parseConfig("base64://bG9nLWxldmVsID0gIndhcm5pbmciCg=="))
}



