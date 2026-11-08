package structs

import (
	"time"
)

type ContentType string
type PlatformType string

const (
	Youtube PlatformType = "YOUTUBE"
)

const (
	Video   ContentType = "VIDEO"
	Channel ContentType = "CHANNEL"
	Shorts  ContentType = "SHORTS"
)

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

//
type BrowseMetadata struct {
	LastCrawledAt time.Time
	LastChannelId ContentId
	WorkerType WorkerType
}

type ContentId struct {
	Id       string      // tenant pattern
	Type     ContentType // блять сука ебал нету нахуй enum-амов.
	Platform PlatformType
}
