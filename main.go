package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/google/shlex"
	"github.com/nlopes/slack"
	"go.spiff.io/wadjet/pkg/reqrep"
)

func main() {
	var listen = ":8080"
	handler := &Handler{}

	flag.StringVar(&listen, "listen", listen, "HTTP listen `address`.")
	flag.StringVar(&handler.slackSignature, "slack-signing-secret", "", "The Slack signing `secret`.")

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/slack/slash", reqrep.AllowPOST(handler))

	if err := http.ListenAndServe(listen, mux); err != nil {
		log.Fatalf("Server error: %+v", err)
	}
}

type Handler struct {
	slackSignature string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Capture the request body to check the signature.
	ensure := func() error { return nil }
	if h.slackSignature != "" {
		verifier, err := slack.NewSecretsVerifier(req.Header, h.slackSignature)
		if err != nil {
			reqrep.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		req.Body = ioutil.NopCloser(io.TeeReader(req.Body, &verifier))
		ensure = verifier.Ensure
	}

	// Parse the slash command from the form body.
	cmd, err := slack.SlashCommandParse(req)
	if err != nil {
		log.Printf("Error parsing request: %+v", err)
		reqrep.Error(w, http.StatusBadRequest, "invalid slash request")
		return
	}

	// Check that the body hashes correctly.
	if err := ensure(); err != nil {
		log.Printf("Error validating request: %+v", err)
		reqrep.Error(w, http.StatusUnauthorized, "signature does not match request")
		return
	}

	// Parse arguments to the command.
	args, err := shlex.Split(cmd.Text)
	if err != nil {
		reqrep.Errorf(w, http.StatusBadRequest, "unable to parse command arguments: %v", err)
		return
	}

	// Look up the command to run.
	cmdName := strings.TrimPrefix(cmd.Command, "/")
	prog, ok := Commands[cmdName]
	if !ok {
		reqrep.Errorf(w, http.StatusBadRequest, "unrecognized command %q", cmdName)
		return
	}

	// Allocate a flag set to parse the args from earlier.
	flags := flag.NewFlagSet(cmdName, flag.ContinueOnError)

	// Capture command output in a buffer.
	var buf bytes.Buffer
	flags.SetOutput(&buf)

	// Run the command.
	msg, err := prog(req.Context(), flags, args)
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		// Exit due to an error.
		reqrep.Errorf(w, http.StatusInternalServerError, "error running command %q: %+v", cmd.Command, err)
		return
	}

	if buf.Len() > 0 && msg == nil {
		msg = &slack.Msg{
			Text:         buf.String(),
			ResponseType: slack.ResponseTypeEphemeral,
		}
	}

	// If the buffer is empty, don't write JSON. The command output may be
	// deffered.
	if msg == nil {
		reqrep.Code(w, http.StatusOK, "")
		return
	}

	// Write response text.
	_ = reqrep.JSON(w, http.StatusOK, msg)
}

type Command func(ctx context.Context, flags *flag.FlagSet, args []string) (*slack.Msg, error)

var Commands = map[string]Command{
	"test": TestCommand,
}

func TestCommand(ctx context.Context, flags *flag.FlagSet, args []string) (*slack.Msg, error) {
	if err := flags.Parse(args); err != nil {
		return nil, err
	}
	fmt.Fprintf(flags.Output(), "Args: %# v", flags.Args())
	return nil, nil
}
