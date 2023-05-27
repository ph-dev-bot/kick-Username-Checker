package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/fatih/color"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

func printUI() {
	fmt.Println("\033[32m _  _____ ___ _  __  _   _      ___")
	fmt.Println("| |/ /_ _/ __| |/ / | | | |___ / __|")
	fmt.Println("| ' < | | (__| ' <  | |_| |___| (__")
	fmt.Println("|_|\\_\\___\\___|_|\\_\\  \\___/     \\___|")
	fmt.Println("\033[0m")
}

func checkUsername(username string) bool {
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(tls_client.Chrome_105),
		tls_client.WithNotFollowRedirects(),
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		log.Println(err)
		return checkUsername(username)
	}

	data := struct {
		Username string `json:"username"`
	}{
		Username: username,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return checkUsername(username)
	}

	req, err := http.NewRequest(http.MethodPost, "https://kick.com/api/v1/signup/verify/username", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println(err)
		return checkUsername(username)
	}

	req.Header = http.Header{
		"Host":            {"kick.com"},
		"Content-Type":    {"application/json"},
		"Accept":          {"application/json"},
		"Accept-Language": {"fr-FR"},
		"User-Agent":      {"Kick/40 CFNetwork/1404.0.5 Darwin/22.3.0"},
		http.HeaderOrderKey: {
			"Host",
			"Content-Type",
			"Accept",
			"Accept-Language",
			"User-Agent",
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return checkUsername(username)
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case 204:
		color.New(color.FgGreen).Printf("[+] Available: %s \n", username)
		return true
	case 422:
		color.New(color.FgRed).Printf("[-] Taken or Invalid: %s \n", username)
		return false
	case 403:
		color.New(color.FgYellow).Println("Cloudflare error and retries")
		return checkUsername(username)
	default:
		log.Printf("Unhandled response code: %d", resp.StatusCode)
	}
	return false

}

func writeInFile(username string) error {
	outputFile, err := os.OpenFile("output.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	_, err = outputFile.WriteString(username + "\n")
	if err != nil {
		return err
	}

	return nil
}

func main() {
	printUI()

	file, err := os.Open("usernames.txt")
	if err != nil {
		fmt.Println("Error opening :", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var usernames []string
	for scanner.Scan() {
		usernames = append(usernames, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file :", err)
		return
	}

	validUsernames := make(chan string)

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter the number of threads to use: ")

	input, _ := reader.ReadString('\n')
	threadCount, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		fmt.Println("Invalid input for thread count")
		return
	}

	if threadCount > len(usernames) {
		threadCount = len(usernames)
		fmt.Printf("Reducing thread count to match the number of usernames: %d\n", threadCount)
	}

	var wg sync.WaitGroup
	wg.Add(threadCount)

	for i := 0; i < threadCount; i++ {
		go func() {
			defer wg.Done()

			for {
				select {
				case username, ok := <-validUsernames:
					if !ok {
						return
					}
					if checkUsername(username) {
						writeInFile(username)
					}
				default:
					return
				}
			}
		}()
	}

	for _, username := range usernames {
		validUsernames <- username
	}
	close(validUsernames)

	wg.Wait()

	fmt.Printf("Done! Valid user names have been saved in output.txt\nUsername found: %d", len(validUsernames))
}
