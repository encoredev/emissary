package emissaryproto

//go:generate protoc --go_out=.. emissary.proto

// EmissaryServerVersion is a var so we can change it at compile time
var EmissaryServerVersion = "devel"

// EmissaryServer is a constant and allows the client to sanity check it's connected to the correct thing
const EmissaryServer = "emissary-server"

// ProtocolVersion allows us forwards compatibility if we redesign the protocol in the future
//
// Versions:
// 1 = SOCKS5 proxy server is available after the connection message is sent by the server
const ProtocolVersion = 1

const NonceSize = 32
