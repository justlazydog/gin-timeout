package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	timeout "github.com/justlazydog/gin-timeout"
)

func main() {
	r := gin.Default()
	r.GET("/hello", timeout.New(
		timeout.WithTimeout(time.Second),
		timeout.WithResponseCode(http.StatusRequestTimeout),
		timeout.WithResponseMsg("request has timeout"),
	), func(c *gin.Context) {
		time.Sleep(3 * time.Second)
		c.JSON(200, "hello")
	})

	r.Run(":8000")
}
