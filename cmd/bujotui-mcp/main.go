package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/studiowebux/bujotui/internal/config"
	"github.com/studiowebux/bujotui/internal/mcp"
	"github.com/studiowebux/bujotui/internal/service"
	"github.com/studiowebux/bujotui/internal/storage"
)

var version = "dev"

func main() {
	logFile := flag.String("logfile", "", "write logs to file instead of stderr")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	logOut := os.Stderr
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) // #nosec G304
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		logOut = f
	}

	log.SetOutput(logOut)
	log.SetPrefix("[bujotui-mcp] ")

	configDir := config.DefaultConfigDir()
	dataDir := config.DefaultDataDir()
	if dir := os.Getenv("BUJOTUI_DIR"); dir != "" {
		configDir = dir
		dataDir = dir
	}

	cfg, err := config.Load(configDir, dataDir)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	store, err := storage.NewStore(cfg)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	svc := service.NewEntryService(store, cfg)
	colSvc := service.NewCollectionService(store)
	habSvc := service.NewHabitService(store)
	futSvc := service.NewFutureLogService(store, cfg)
	handler := mcp.NewHandler(svc, colSvc, habSvc, futSvc)
	transport := mcp.NewTransport(os.Stdin, os.Stdout)

	log.Printf("starting bujotui MCP server %s", version)

	for {
		msg, err := transport.Read()
		if err != nil {
			log.Printf("read error: %v", err)
			return
		}

		resp := handleMessage(handler, msg, version)
		if resp != nil {
			if err := transport.Write(resp); err != nil {
				log.Printf("write error: %v", err)
				return
			}
		}
	}
}

func handleMessage(h *mcp.Handler, msg *mcp.Message, ver string) *mcp.Message {
	if msg.ID == nil {
		return nil // notification, ignore
	}

	switch msg.Method {
	case "initialize":
		result := mcp.InitializeResult{
			ProtocolVersion: "2024-11-05",
			ServerInfo:      mcp.ServerInfo{Name: "bujotui-mcp", Version: ver},
			Capabilities:    mcp.Capabilities{Tools: &mcp.ToolsCapability{}},
		}
		resp, err := mcp.NewResponse(msg.ID, result)
		if err != nil {
			return mcp.NewErrorResponse(msg.ID, -32603, err.Error())
		}
		return resp

	case "tools/list":
		result := mcp.ToolList()
		resp, err := mcp.NewResponse(msg.ID, result)
		if err != nil {
			return mcp.NewErrorResponse(msg.ID, -32603, err.Error())
		}
		return resp

	case "tools/call":
		var params mcp.ToolCallParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return mcp.NewErrorResponse(msg.ID, -32602, fmt.Sprintf("invalid params: %v", err))
		}
		result := h.HandleToolCall(params.Name, params.Arguments)
		resp, err := mcp.NewResponse(msg.ID, result)
		if err != nil {
			return mcp.NewErrorResponse(msg.ID, -32603, err.Error())
		}
		return resp

	default:
		return mcp.NewErrorResponse(msg.ID, -32601, fmt.Sprintf("method not found: %s", msg.Method))
	}
}
