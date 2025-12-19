// Copyright (C) by Ubaldo Porcheddu

package test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	serverPort    = "45678"
	bindHost      = "0.0.0.0"   // Must be 0.0.0.0 to enable Discovery service
	clientHost    = "127.0.0.1" // We connect via loopback
	serverURL     = "http://" + clientHost + ":" + serverPort
	testPassword  = "testsecret"
	buildName     = "taz_test_bin"
	testRootFiles = "temp_test_files"
)

// TestMain handles compilation and global cleanup
func TestMain(m *testing.M) {
	// 1. Build the binary from ../app
	fmt.Println("[Setup] Building application...")
	buildCmd := exec.Command("go", "build", "-o", buildName, "../app")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Printf("Failed to build app: %v\n", err)
		os.Exit(1)
	}

	// 2. Create a temp directory for file operations
	if err := os.MkdirAll(testRootFiles, 0755); err != nil {
		fmt.Printf("Failed to create temp root: %v\n", err)
		os.Exit(1)
	}

	// 3. Run tests
	exitCode := m.Run()

	// 4. Cleanup
	fmt.Println("[Teardown] Cleaning up...")
	os.Remove(buildName)
	os.RemoveAll(testRootFiles)
	os.Exit(exitCode)
}

// TestAppSuite runs the server once and executes all sub-tests against it
func TestAppSuite(t *testing.T) {
	// Start the server in the background
	cmd := exec.Command("./"+buildName,
		"--web-port", serverPort,
		"--web-host", bindHost, // 0.0.0.0 allows discovery to start
		"--root", testRootFiles,
		"--password", testPassword,
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	
	// Ensure server is killed when this test function exits
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// Wait for server to be ready
	if !waitForServer(t) {
		t.Fatal("Server failed to start within timeout")
	}

	// Create a shared client with a cookie jar to persist login session across subtests
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// --- Run Sub-Tests sequentially ---
	// 1. check status
	t.Run("StatusEndpoint", func(t *testing.T) { testStatusEndpoint(t, client) })
	
	// 2. check unauthorized access
	t.Run("UnauthorizedAccess", func(t *testing.T) { testUnauthorizedAccess(t) })
	
	// 3. check udp discovery (requires server running on 0.0.0.0)
	t.Run("DiscoveryUDP", func(t *testing.T) { testDiscoveryUDP(t) })
	
	// 4. login (persists cookie in jar)
	t.Run("LoginFlow", func(t *testing.T) { testLoginFlow(t, client) })
	
	// 5. file upload (requires login from previous step)
	t.Run("FileUpload", func(t *testing.T) { testFileUpload(t, client) })
	
	// 6. bbs posting (requires login)
	t.Run("BBSFunctionality", func(t *testing.T) { testBBSFunctionality(t, client) })
}

func waitForServer(t *testing.T) bool {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", clientHost+":"+serverPort, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// --- Implementation of Sub-Tests ---

func testStatusEndpoint(t *testing.T, client *http.Client) {
	resp, err := client.Get(serverURL + "/status")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "uptime") {
		t.Errorf("Status response missing 'uptime': %s", string(body))
	}
}

func testUnauthorizedAccess(t *testing.T) {
	// Use a fresh client without cookies
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(serverURL + "/")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	
	// App allows read-only access (200 OK) even without login, 
	// but UI should show login form, not authenticated UI.
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK for read-only access, got %d", resp.StatusCode)
	}
}

func testLoginFlow(t *testing.T, client *http.Client) {
	// 1. Attempt Wrong Password
	form := url.Values{}
	form.Add("password", "wrongpass")
	
	resp, err := client.PostForm(serverURL+"/login", form)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Verify NO session cookie in jar
	u, _ := url.Parse(serverURL)
	cookies := client.Jar.Cookies(u)
	hasAuth := false
	for _, c := range cookies {
		if c.Name == "taz_auth" && c.Value != "" {
			hasAuth = true
		}
	}
	if hasAuth {
		t.Error("Received auth cookie with wrong password")
	}

	// 2. Attempt Correct Password
	form.Set("password", testPassword)
	resp, err = client.PostForm(serverURL+"/login", form)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Verify Session Cookie
	cookies = client.Jar.Cookies(u)
	validCookie := false
	for _, c := range cookies {
		if c.Name == "taz_auth" && c.Value != "" {
			validCookie = true
		}
	}
	if !validCookie {
		t.Error("Failed to receive auth cookie with correct password")
	}

	// 3. Perform a Write Operation (Create Folder) to confirm permissions
	form = url.Values{}
	form.Add("action", "mkdir")
	form.Add("dirname", "test_folder")
	form.Add("path", ".")

	resp, err = client.PostForm(serverURL+"/", form)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Verify directory exists on disk
	expectedDir := filepath.Join(testRootFiles, "test_folder")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Directory was not created on disk at %s", expectedDir)
	}
}

func testDiscoveryUDP(t *testing.T) {
	// Send "TAZ_DISCOVER" to UDP port
	addr, err := net.ResolveUDPAddr("udp", clientHost+":"+serverPort)
	if err != nil {
		t.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))
	
	msg := []byte("TAZ_DISCOVER")
	_, err = conn.Write(msg)
	if err != nil {
		t.Fatal(err)
	}

	// Read response
	buf := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("UDP Discovery read failed: %v", err)
	}

	response := string(buf[:n])
	// Expected format: TAZ_IDENT|<Name>|<Version>
	if !strings.HasPrefix(response, "TAZ_IDENT|") {
		t.Errorf("Invalid discovery response: %s", response)
	}
}

func testFileUpload(t *testing.T, client *http.Client) {
	// Prepare Multipart Upload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file field
	part, err := writer.CreateFormFile("files", "uploaded_test.txt")
	if err != nil {
		t.Fatal(err)
	}
	fileContent := "This is a test file content."
	part.Write([]byte(fileContent))

	// Add required fields
	writer.WriteField("action", "upload")
	writer.WriteField("path", ".")

	writer.Close()

	// Send Request
	req, err := http.NewRequest("POST", serverURL+"/", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Verify file exists on disk
	expectedPath := filepath.Join(testRootFiles, "uploaded_test.txt")
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Uploaded file not found on disk: %v", err)
	}
	if string(content) != fileContent {
		t.Errorf("File content mismatch. Got: %s", string(content))
	}
}

func testBBSFunctionality(t *testing.T, client *http.Client) {
	// Post a Message to BBS
	testMsg := "Hello from Automated Test"
	form := url.Values{}
	form.Set("message", testMsg)

	resp, err := client.PostForm(serverURL+"/bbs", form)
	if err != nil {
		t.Fatalf("Failed to post to BBS: %v", err)
	}
	resp.Body.Close()

	// Read BBS Page to verify message appears
	resp, err = client.Get(serverURL + "/bbs")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	if !strings.Contains(bodyString, testMsg) {
		t.Errorf("BBS did not contain the posted message. Body snippet: %s", bodyString[:200])
	}
}
