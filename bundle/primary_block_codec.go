package bundle

//go:generate codecgen -o primary_block_codecgen.go primary_block_codec.go

// primaryBlock11 is a serialization type with: Fragmentation and CRC
type primaryBlock11 struct {
	_struct bool `codec:",toarray"`

	Version            uint
	BundleControlFlags BundleControlFlags
	CRCType            CRCType
	Destination        EndpointID
	SourceNode         EndpointID
	ReportTo           EndpointID
	CreationTimestamp  CreationTimestamp
	Lifetime           uint
	FragmentOffset     uint
	TotalDataLength    uint
	CRC                []byte
}

// primaryBlock10 is a serialization type with: Fragmentation and no CRC
type primaryBlock10 struct {
	_struct bool `codec:",toarray"`

	Version            uint
	BundleControlFlags BundleControlFlags
	CRCType            CRCType
	Destination        EndpointID
	SourceNode         EndpointID
	ReportTo           EndpointID
	CreationTimestamp  CreationTimestamp
	Lifetime           uint
	FragmentOffset     uint
	TotalDataLength    uint
}

// primaryBlock09 is a serialization type with: CRC and no Fragmentation
type primaryBlock09 struct {
	_struct bool `codec:",toarray"`

	Version            uint
	BundleControlFlags BundleControlFlags
	CRCType            CRCType
	Destination        EndpointID
	SourceNode         EndpointID
	ReportTo           EndpointID
	CreationTimestamp  CreationTimestamp
	Lifetime           uint
	CRC                []byte
}

// primaryBlock08 is a serialization type with: no CRC and no Fragmentation
type primaryBlock08 struct {
	_struct bool `codec:",toarray"`

	Version            uint
	BundleControlFlags BundleControlFlags
	CRCType            CRCType
	Destination        EndpointID
	SourceNode         EndpointID
	ReportTo           EndpointID
	CreationTimestamp  CreationTimestamp
	Lifetime           uint
}
