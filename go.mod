module github.com/dtn7/dtn7-go

require (
	github.com/AndreasBriese/bbloom v0.0.0-20190825152654-46b345b51c96 // indirect
	github.com/BurntSushi/toml v0.3.1
	github.com/RyanCarrier/dijkstra v0.0.0-20190613134106-3f5a38e7002e
	github.com/dtn7/cboring v0.1.5
	github.com/dtn7/rf95modem-go v0.2.0
	github.com/felixge/tcpkeepalive v0.0.0-20160804073959-5bb0b2dea91e
	github.com/fsnotify/fsnotify v1.4.7
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/gorilla/websocket v1.4.1
	github.com/hashicorp/go-multierror v1.0.0
	github.com/howeyc/crc16 v0.0.0-20171223171357-2b2a61e366a6
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/schollz/peerdiscovery v1.4.0
	github.com/sirupsen/logrus v1.4.2
	github.com/timshannon/badgerhold v0.0.0-20190415130923-192650dd187a
	github.com/ulikunitz/xz v0.5.6
	golang.org/x/net v0.0.0-20190724013045-ca1201d0de80 // indirect
	golang.org/x/sys v0.0.0-20191210023423-ac6580df4449 // indirect
)

exclude (
	github.com/RyanCarrier/dijkstra-1 v0.0.0-20170512020943-0e5801a26345
	github.com/albertorestifo/dijkstra v0.0.0-20160910063646-aba76f725f72
	github.com/mattomatic/dijkstra v0.0.0-20130617153013-6f6d134eb237
)

go 1.13
