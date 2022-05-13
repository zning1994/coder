package provisionersdk

import (
	"github.com/coder/coder/provisionersdk/proto"
)

func LogToParseStream(stream proto.DRPCProvisioner_ParseStream, l proto.LogLevel, msg string) error {
	log := proto.Parse_Response{}
	log.Type = &proto.Parse_Response_Log{
		Log: &proto.Log{
			Level: l,
			Output: msg,
		},
	}
	return stream.Send(&log)
}
