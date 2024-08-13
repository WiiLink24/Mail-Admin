package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/WiiLink24/nwc24"
	"github.com/gin-gonic/gin"
)

func generateBoundary() string {
	source := rand.NewSource(time.Now().Unix())
	val := rand.New(source)
	return fmt.Sprintf("%s/%d", time.Now().Format("200601021504"), val.Intn(8999999)+1000000)
}

func validateFriendCode(strId string) bool {
	if len(strId) != 16 {
		// All Wii Numbers are 16 characters long.
		return false
	}

	id, err := strconv.ParseInt(strId, 10, 64)
	if err != nil {
		// Not an integer value, therefore not an ID
		return false
	}

	wiiNumber := nwc24.LoadWiiNumber(uint64(id))
	if !wiiNumber.CheckWiiNumber() {
		// Invalid Wii Number (crc is invalid)
		return false
	}

	return !(wiiNumber.GetHollywoodID() == 0x0403AC68)
}

func uploadToGenerator(c *gin.Context, source string, destination string) {
	file, header, err := c.Request.FormFile(source)
	if err != nil {
		log.Printf("Failed to get uploaded file: %v", err)
		return
	}
	defer file.Close()

	// Get the filename
	filename := header.Filename
	fmt.Println("Received file:", filename)

	// Ensure the generator/input directory exists
	err = os.MkdirAll("generator/input", os.ModePerm)
	if err != nil {
		log.Printf("Failed to create directory: %v", err)
		return
	}

	// Create a file for the letterhead
	out, err := os.Create("generator/input/" + destination)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	// Write the uploaded file to the created file
	_, err = io.Copy(out, file)
	if err != nil {
		log.Printf("Failed to write letterhead data: %v", err)
	}
}

func generateLetterhead() (string, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer

	// Run generate.sh
	cmd := exec.Command("sh", "generator/generate.sh")
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	log.Printf("Running command and waiting for it to finish...")
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run generate.sh: %v: %s", err, stderr.String())
	}

	// Read the data from generator/output/letterhead.txt
	letterheadData, err := ioutil.ReadFile("generator/output/letterhead.txt")
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// Convert the data to a string and store it in a variable
	letterheadContent := string(letterheadData)
	return letterheadContent, nil
}