package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateMessagePage(c *gin.Context) {
	c.HTML(http.StatusOK, "send_message.html", nil)
}

func ClearMessagesPage(c *gin.Context) {
	c.HTML(http.StatusOK, "clear_messages.html", nil)
}

func MiscPage(c *gin.Context) {
	c.HTML(http.StatusOK, "misc.html", nil)
}


