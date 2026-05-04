package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func extractStatusFromDynamicResponse(reader *bufio.Reader, source string) (int, error) {
	rawHeader, _, err := reader.ReadLine()
	if err != nil {
		log.Println("Unable to read header in dynamic output from " + source + ".")
		return 0, err
	}
	header := string(rawHeader)
	logErrorMsg := "Unable to parse first line of dynamic output from " +
	               source +
		       " as valid Gemini response header.  Line was: " +
		       header
	header_fields := strings.Fields(header)
	if len(header_fields) == 0 {
		log.Println(logErrorMsg)
		return 0, errors.New("")
	}
	status, err := strconv.Atoi(header_fields[0])
	if err != nil {
		log.Println(logErrorMsg)
		return 0, err
	}
	if status < 10 || status > 70 {
		log.Println(logErrorMsg)
		return 0, errors.New("")
	}

	return status, nil
}

func handleCGI(config SysConfig, path string, cgiPath string, URL *url.URL, logEntry *LogEntry, conn net.Conn, isKepler bool) {
	// Find the shortest leading part of path which maps to an executable file.
	// Call this part scriptPath, and everything after it pathInfo.
	components := strings.Split(path, "/")
	scriptPath := ""
	pathInfo := ""
	matched := false
	for i := 0; i <= len(components); i++ {
		scriptPath = strings.Join(components[0:i], "/")
		pathInfo = strings.Join(components[i:], "/")
		if !strings.HasPrefix(scriptPath, cgiPath) {
			continue
		}
		info, err := os.Stat(scriptPath)
		if err != nil {
			break
		} else if info.IsDir() {
			continue
		} else if info.Mode().Perm()&0555 == 0555 {
			matched = true
			break
		}
	}
	// If we didn't find a match, give up and let this request be handled as
	// if it were a static file
	if !matched {
		return
	}

	// Prepare environment variables
	vars := prepareCGIVariables(config, URL, conn, scriptPath, pathInfo, isKepler)

	// Spawn process
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.CGITimeLimit)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, scriptPath)
	cmd.Env = []string{}
	for key, value := range vars {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	response, err := cmd.Output()

	if ctx.Err() == context.DeadlineExceeded {
		log.Println("Terminating CGI process " + path + " due to exceeding runtime limit.")
		conn.Write([]byte("42 CGI process timed out!\r\n"))
		logEntry.Status = 42
		return
	}
	if err != nil {
		log.Println("Error running CGI program " + path + ": " + err.Error())
		if err, ok := err.(*exec.ExitError); ok {
			log.Println("↳ stderr output: " + string(err.Stderr))
		}
		conn.Write([]byte("42 CGI error!\r\n"))
		logEntry.Status = 42
		return
	}

	// Extract response header
	responseString := string(response)
	if len(responseString) == 0 {
		log.Println("Received no response from CGI process " + path)
		conn.Write([]byte("42 CGI error!\r\n"))
		logEntry.Status = 42
		return
	}

	responseReader := bufio.NewReader(strings.NewReader(string(response)))
	status, err := extractStatusFromDynamicResponse(responseReader, path)
	if err != nil {
		conn.Write([]byte("42 CGI error!\r\n"))
		logEntry.Status = 42
		return
	} else {
		logEntry.Status = status
	}

	// Write response
	conn.Write(response)
}

func handleSCGI(URL *url.URL, scgiPath string, scgiSocket string, config SysConfig, logEntry *LogEntry, conn net.Conn, isKepler bool) {

	// Connect to socket
	socket, err := net.Dial("unix", scgiSocket)
	if err != nil {
		log.Println("Error connecting to SCGI socket " + scgiSocket + ": " + err.Error())
		conn.Write([]byte("42 Error connecting to SCGI service!\r\n"))
		logEntry.Status = 42
		return
	}
	defer socket.Close()

	// Send variables
	vars := prepareSCGIVariables(config, URL, scgiPath, conn, isKepler)
	length := 0
	for key, value := range vars {
		length += len(key)
		length += len(value)
		length += 2
	}
	socket.Write([]byte(strconv.Itoa(length) + ":"))
	for key, value := range vars {
		socket.Write([]byte(key + "\x00"))
		socket.Write([]byte(value + "\x00"))
	}
	socket.Write([]byte(","))

	// Read and relay response
	responseReader := bufio.NewReader(socket)
	status, err := extractStatusFromDynamicResponse(responseReader, scgiSocket)
	if err != nil {
		conn.Write([]byte("42 SCGI error!\r\n"))
		logEntry.Status = 42
		return
	} else {
		logEntry.Status = status
	}

	buffer := make([]byte, 1027)
	for {
		n, err := responseReader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				// Err
				log.Println("Error reading from SCGI socket " + scgiSocket + ": " + err.Error())
				break
			}
		}
		// Send to client
		conn.Write(buffer[:n])
	}
}

func prepareCGIVariables(config SysConfig, URL *url.URL, conn net.Conn, script_path string, path_info string, isKepler bool) map[string]string {
	vars := prepareGatewayVariables(config, URL, conn, isKepler)
	vars["GATEWAY_INTERFACE"] = "CGI/1.1"
	vars["SCRIPT_PATH"] = script_path
	vars["PATH_INFO"] = path_info
	return vars
}

func prepareSCGIVariables(config SysConfig, URL *url.URL, scgiPath string, conn net.Conn, isKepler bool) map[string]string {
	vars := prepareGatewayVariables(config, URL, conn, isKepler)
	vars["SCGI"] = "1"
	vars["CONTENT_LENGTH"] = "0"
	vars["SCRIPT_PATH"] = scgiPath
	vars["PATH_INFO"] = URL.Path[len(scgiPath):]
	return vars
}

func prepareGatewayVariables(config SysConfig, URL *url.URL, conn net.Conn, isKepler bool) map[string]string {
	vars := make(map[string]string)
	vars["QUERY_STRING"] = URL.RawQuery
	vars["REQUEST_METHOD"] = ""
	vars["SERVER_NAME"] = config.Hostname
	vars["SERVER_PORT"] = strconv.Itoa(config.Port)
        if (isKepler) {
	  vars["SERVER_PROTOCOL"] = "kepler"
        } else {
	  vars["SERVER_PROTOCOL"] = "gemini"
        }
	vars["SERVER_SOFTWARE"] = "MOLLY_BROWN"

	host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	vars["REMOTE_ADDR"] = host

	// Add TLS variables
	var tlsConn (*tls.Conn) = conn.(*tls.Conn)
	connState := tlsConn.ConnectionState()
	//	vars["TLS_CIPHER"] = CipherSuiteName(connState.CipherSuite)

	// Add client cert variables
	clientCerts := connState.PeerCertificates
	if len(clientCerts) > 0 {
		cert := clientCerts[0]
		vars["TLS_CLIENT_HASH"] = getCertFingerprint(cert)
		vars["TLS_CLIENT_ISSUER"] = cert.Issuer.String()
		vars["TLS_CLIENT_ISSUER_CN"] = cert.Issuer.CommonName
		vars["TLS_CLIENT_SUBJECT"] = cert.Subject.String()
		vars["TLS_CLIENT_SUBJECT_CN"] = cert.Subject.CommonName
		// To make it easier to detect when a cert is present
		vars["AUTH_TYPE"] = "Certificate"
	}
	return vars
}
