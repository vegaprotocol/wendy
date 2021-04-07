package wendy

import "google.golang.org/protobuf/proto"

func MustMarshal(msg proto.Message) []byte {
	bz, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return bz
}

func MustUnmarshal(b []byte, m proto.Message) {
	if err := proto.Unmarshal(b, m); err != nil {
		panic(err)
	}
}
