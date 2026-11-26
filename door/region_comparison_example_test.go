package door

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

func Example_compareRegions() {
	here, _ := os.Getwd()
	env, _ := NewEnv(Given{Context: context.Background(), Directory: here,
		Now: time.Now, Environment: func(string) (string, bool) { return "", false }})
	var said bytes.Buffer
	Tool(env).Run([]string{"compare-regions", "--help", "--dry-run"}, &said, &said)
	for _, line := range strings.Split(said.String(), "\n") {
		if strings.Contains(line, "--complete-inputs") {
			fmt.Println(strings.TrimSpace(line))
		}
	}
	// Output: --complete-inputs         assert the named inputs are complete and stable
}
