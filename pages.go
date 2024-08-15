package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateMessagePage(c *gin.Context) {
	c.HTML(http.StatusOK, "send_message.html", gin.H{
		"Title": "Send Message | WiiLink Mail",
	})
}

func ClearMessagesPage(c *gin.Context) {
	c.HTML(http.StatusOK, "inbound.html", nil)
}

func MiscPage(c *gin.Context) {
	c.HTML(http.StatusOK, "misc.html", gin.H{
		"Title": "Miscellaneous | WiiLink Mail",
	})
}
