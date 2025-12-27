package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func CreateMessagePage(c *gin.Context) {
	c.HTML(http.StatusOK, "send_message.html", gin.H{
		"Title": "Send Message | WiiLink Mail",
	})
}

func CreateMessageMultiLangPage(c *gin.Context) {
	c.HTML(http.StatusOK, "send_message_multilang.html", gin.H{
		"Title": "Send Multi-language Message | WiiLink Mail",
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

// Get and parse announcement files
func AnnouncementsPage(c *gin.Context) {
	type LangAnn struct {
		Code    string
		ID      string
		AltName string
		Message string
	}

	langMap := map[string]string{"0": "Japanese", "1": "English", "2": "German", "3": "French", "4": "Spanish", "5": "Italian", "6": "Dutch", "7": "Other"}

	anns := make([]LangAnn, 0)

	// list subfolders under assets path
	entries, _ := os.ReadDir(config.AssetsPath)

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if _, ok := langMap[name]; !ok {
			continue
		}

		annPath := filepath.Join(config.AssetsPath, name, "announcement")
		la := LangAnn{Code: name, ID: langMap[name]}
		if _, err := os.Stat(annPath); err == nil {
			alt, msg, perr := parseAnnouncementFile(annPath)
			if perr == nil {
				la.AltName = alt
				la.Message = msg
			}
		}
		anns = append(anns, la)
	}

	hasActive := false
	for _, a := range anns {
		if a.AltName != "" {
			hasActive = true
			break
		}
	}

	c.HTML(http.StatusOK, "announcements.html", gin.H{
		"Title":     "Announcements | WiiLink Mail",
		"Anns":      anns,
		"HasActive": hasActive,
	})
}

// Delete announcement file
func StopAnnouncement(c *gin.Context) {
	code := c.PostForm("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "missing code"})
		return
	}
	annPath := filepath.Join(config.AssetsPath, code, "announcement")
	if _, err := os.Stat(annPath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "announcement not found"})
		return
	}
	if err := os.Remove(annPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "stopped"})
}
