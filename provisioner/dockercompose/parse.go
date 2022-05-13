package dockercompose

import (
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/xerrors"

	"github.com/coder/coder/provisionersdk"
	"github.com/coder/coder/provisionersdk/proto"
)

//const validShellVariableRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"

var variableRegexp = regexp.MustCompile("\\${([a-zA-Z0-9_]+)[^}]*}|\\$([a-zA-Z0-9]+)")
var coderEnvVars = []string{
	"CODER_AGENT_URL",
	"CODER_WORKSPACE_TRANSITION",
	"CODER_WORKSPACE_NAME",
	"CODER_WORKSPACE_OWNER",
	"CODER_WORKSPACE_ID",
	"CODER_WORKSPACE_OWNER_ID",
}

type dockerComposeConfig map[string]interface{}

func (p *provisionerServer) Parse(req *proto.Parse_Request, stream proto.DRPCProvisioner_ParseStream) error {
	p.log.Info(stream.Context(), "got dockercompose parse request")
	defer func() {
		_ = stream.CloseSend()
	}()

	dcBytes, err := os.ReadFile(req.GetDirectory() + "/docker-compose.yml")
	if err != nil {
		return xerrors.Errorf("unable to read docker-compose.yml: %w", err)
	}
	provisionersdk.LogToParseStream(stream, proto.LogLevel_INFO, "read docker-compose.yml")

	params, err := extractParameters(dcBytes)
	if err != nil {
		return xerrors.Errorf("invalid parameter format in docker-compose.yml: %w", err)
	}
	complete := &proto.Parse_Response{
		Type: &proto.Parse_Response_Complete{
			Complete: &proto.Parse_Complete{
				ParameterSchemas: params,
			},
		},
	}
	stream.Send(complete)
	return nil
}

// type parser struct {
// 	stack  []string
// 	params []*proto.ParameterSchema
// }

// func (p *parser) process(r rune) error {
// 	switch  {
// 	case r == '$':
// 		return p.dollarSign()
// 	case r == '{':
// 		return p.openCurlyBrace()
// 	case r == '}':
// 		return p.closeParen()
// 	case r == ':':
// 		return p.colon()
// 	case r == '-' || r == '?':
// 		return p.dashOrQuestionMark(string(r))
// 	case isValidShellVariableRune(r):
// 		return p.shellVariableRune(r)
// 	default:
// 		return p.otherRune(r)
// 	}
// }

// func (p *parser) push(s string) {
// 	p.stack = append(p.stack, s)
// }

// func (p *parser) peek() string {
// 	return p.stack[len(p.stack)-1]
// }

// func (p *parser) pop() string {
// 	v := p.peek()
// 	p.stack = p.stack[:len(p.stack)-1]
// 	return v
// }

// func (p *parser) openCurlyBrace() error {
// 	if len(p.stack) == 0 {
// 		return nil
// 	}
// 	switch p.peek() {
// 	case "$":
// 		p.push("{")
// 		return nil
// 	default:
// 		return xerrors.New("invalid {")
// 	}
// }

// func (p *parser) dollarSign() error {
// 	if len(p.stack) == 0 {
// 		p.push("$")
// 		return nil
// 	}
// 	switch p.peek() {
// 	case "$":
// 		_ = p.pop()
// 		return nil
// 	case "?":
// 		p.push("$")
// 		return nil
// 	case "-":
// 		p.push("$")
// 		return nil
// 	default:
// 		return xerrors.New("invalid $")
// 	}
// }

// // closeParen attempts to pop from the stack back to a { and extract the variable
// func (p *parser) closeParen() error {
// 	if len(p.stack) == 0 {
// 		return nil
// 	}
// 	name := ""
// 	for len(p.stack) > 0 {
// 		s := p.pop()
// 		switch s {
// 		case "?":  // ${VARNAME?error message}
// 			continue
// 		case "-": // ${VARNAME-default}
// 			continue
// 		case ":": // ${VARNAME:?error message}
// 			continue
// 		case "$": // ${VARNAME-$DEFAULT_VAR}
// 			if name != "" {
// 				p.addParam(name)
// 				name = ""
// 				continue
// 			}
// 			return xerrors.New("invalid }")
// 		case "{":
// 			if name != "" {
// 				p.addParam(name)
// 				return nil
// 			}
// 			return xerrors.New("invalid }")
// 		}
// 	}
// 	return xerrors.New("invalid }")
// }

// func (p *parser) dashOrQuestionMark(s string) error {
// 	if len(p.stack) == 0 {
// 		return nil
// 	}
// 	if k := p.peek(); !isValidShellVariableRune(rune(k[len(k)-1])) && k != ":" {
// 		// if we're in a variable definition, -/? can only come after a variable name or a colon
// 		// like ${VARNAME:-default} or ${VARNAME-default}
// 		return xerrors.Errorf("invalid %s", s)
// 	}
// 	p.push(s)
// 	return nil
// }

// func (p *parser) colon() error {
// 	if len(p.stack) == 0 {
// 		return nil
// 	}
// 	if k := p.peek(); !isValidShellVariableRune(rune(k[len(k)-1])) {
// 		// if we're in a variable definition, colon can only come after a variable name like
// 		// ${VARNAME:?error message}
// 		return xerrors.New("invalid :")
// 	}
// 	p.push(":")
// 	return nil
// }

// func (p *parser) addParam(name string) {
// 	p.params = append(p.params, &proto.ParameterSchema{
// 		Name: name,
// 		Description: name,
// 		AllowOverrideSource:  true,
// 		RedisplayValue: true,
// 		DefaultDestination: &proto.ParameterDestination{
// 			Scheme: proto.ParameterDestination_ENVIRONMENT_VARIABLE,
// 		},
// 		AllowOverrideDestination: false,
// 	})
// }

// func isValidShellVariableRune(r rune) bool {
// 	for _, c := range []rune(validShellVariableRunes) {
// 		if r == c {
// 			return true
// 		}
// 	}
// 	return false
// }

func isCoderEnvVar(name string) bool {
	for _, e := range coderEnvVars {
		if name == e {
			return true
		}
	}
	return strings.HasPrefix(name, "CODER_AGENT_SCRIPT_")
}

func extractParameters(b []byte) ([]*proto.ParameterSchema, error) {
	if !utf8.Valid(b) {
		return nil, xerrors.New("not valid UTF-8 encoded text")
	}
	matches := variableRegexp.FindAllSubmatch(b, -1)
	params := []*proto.ParameterSchema{}
	for _, m := range matches {
		// regex contains an or with two capturing groups.  Take the nonzero one.
		name := ""
		if len(m[1]) > 0 {
			name = string(m[1])
		}
		if len(m[2]) > 0 {
			name = string(m[2])
		}
		if name == "" {
			return nil, xerrors.Errorf("invalid shell variable %s", string(m[0]))
		}
		if isCoderEnvVar(name) {
			continue
		}
		params = append(params, &proto.ParameterSchema{
			Name:                name,
			Description:         name,
			AllowOverrideSource: true,
			RedisplayValue:      true,
			DefaultDestination: &proto.ParameterDestination{
				Scheme: proto.ParameterDestination_ENVIRONMENT_VARIABLE,
			},
			AllowOverrideDestination: false,
		})
	}
	return params, nil
}
