package redis

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
	"gitlab-odx.oracle.com/odx/functions/api/datastore/internal/datastoretest"
)

const tmpRedis = "redis://%s:%d/"

var (
	redisHost = func() string {
		host := os.Getenv("REDIS_HOST")
		if host == "" {
			host = "127.0.0.1"
		}
		return host
	}()
	redisPort = func() int {
		port := os.Getenv("REDIS_PORT")
		if port == "" {
			port = "6301"
		}
		p, err := strconv.Atoi(port)
		if err != nil {
			panic(err)
		}
		return p
	}()
)

func prepareRedisTest(logf, fatalf func(string, ...interface{})) (func(), func()) {
	timeout := time.After(20 * time.Second)

	for {
		c, err := redis.DialURL(fmt.Sprintf(tmpRedis, redisHost, redisPort))
		if err == nil {
			_, err = c.Do("PING")
			c.Close()
			if err == nil {
				break
			}
		}
		fmt.Println("failed to PING redis:", err)
		select {
		case <-timeout:
			log.Fatal("timed out waiting for redis")
		case <-time.After(500 * time.Millisecond):
			continue
		}
	}
	fmt.Println("redis for test ready")
	return func() {},
		func() {
			tryRun(logf, "stop redis container", exec.Command("docker", "rm", "-fv", "func-redis-test"))
		}
}

func TestDatastore(t *testing.T) {
	_, close := prepareRedisTest(t.Logf, t.Fatalf)
	defer close()

	u, err := url.Parse(fmt.Sprintf(tmpRedis, redisHost, redisPort))
	if err != nil {
		t.Fatal("failed to parse url: ", err)
	}
	ds, err := New(u)
	if err != nil {
		t.Fatal("failed to create redis datastore:", err)
	}

	datastoretest.Test(t, ds)
}

func tryRun(logf func(string, ...interface{}), desc string, cmd *exec.Cmd) {
	var b bytes.Buffer
	cmd.Stderr = &b
	if err := cmd.Run(); err != nil {
		logf("failed to %s: %s", desc, b.String())
	}
}

func mustRun(fatalf func(string, ...interface{}), desc string, cmd *exec.Cmd) {
	var b bytes.Buffer
	cmd.Stderr = &b
	if err := cmd.Run(); err != nil {
		fatalf("failed to %s: %s", desc, b.String())
	}
}