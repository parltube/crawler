package structs

import "time"

// base type for metadata, timestamp and generics interface
type BasicType struct {
	TimeStamp time.Time
}

// video metadata. only video may contain comments
type VideoMetaData struct {
	BasicType
}

type ChannelMetadata struct {
	BasicType
}

type CommentsMetadata struct {
	BasicType
}

// player metdata mostly?
type PlayerMetadata struct {
	BasicType
}

type BrowseMetadata struct {
	BasicType
}
