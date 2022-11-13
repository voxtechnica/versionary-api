package image

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"
)

var (
	ctx     = context.Background()
	service = Service{
		EntityType: "Image",
		Bucket:     NewMemBucket(NewBucket(nil, "test")),
		Table:      NewMemTable(NewTable(nil, "test")),
	}
)

func TestImageAnalysis(t *testing.T) {
	expect := assert.New(t)

	// Fetch and analyze a JPEG of a bear
	id := tuid.NewID()
	at, _ := id.Time()
	bear := Image{
		ID:        id.String(),
		CreatedAt: at,
		VersionID: id.String(),
		UpdatedAt: at,
		Title:     "Placeholder Image of a Bear",
		AltText:   "random bear",
		SourceURI: "https://placebear.com/640/320.jpg",
	}
	imgBear, err := service.FetchSourceURI(ctx, bear.SourceURI)
	expect.NoError(err, "fetching bear image")
	expect.NotNil(imgBear, "bear image bytes exist")
	bear, err = service.Analyze(bear, imgBear)
	expect.NoError(err, "analyzing bear image")
	expect.Equal(JPEG, bear.MediaType, "media type is correct")
	expect.Equal(bear.ID+bear.FileExt(), bear.FileName, "file name is correct")
	expect.Equal(640, bear.Width, "width is correct")
	expect.Equal(320, bear.Height, "height is correct")
	expect.Equal(2.0, bear.AspectRatio, "aspect ratio is correct")

	// Analyze a small PNG with transparent background
	id = tuid.NewID()
	at, _ = id.Time()
	stack64 := Image{
		ID:             id.String(),
		CreatedAt:      at,
		VersionID:      id.String(),
		UpdatedAt:      at,
		Title:          "Stack Icon 64x64",
		AltText:        "Stack Icon 64x64",
		SourceFileName: "testdata/stack.64.png",
	}
	img64, err := service.FetchSourceFile(stack64.SourceFileName)
	expect.NoError(err, "no error fetching source image")
	expect.NotNil(img64, "image bytes exist")
	stack64, err = service.Analyze(stack64, img64)
	expect.NoError(err, "no error analyzing image")
	expect.Equal(int64(1015), stack64.FileSize, "file size is correct")
	expect.Equal(PNG, stack64.MediaType, "media type is correct")
	expect.Equal("93ce174c4a9c1bd44e73e15f0799e746", stack64.MD5Hash, "MD5 hash is correct")
	expect.Equal("0", stack64.PHash, "pHash is correct")
	expect.Equal(64, stack64.Width, "width is correct")
	expect.Equal(64, stack64.Height, "height is correct")
	expect.Equal(1.0, stack64.AspectRatio, "aspect ratio is correct")

	// Analyze a large original JPEG image
	id = tuid.NewID()
	at, _ = id.Time()
	jpeg := Image{
		ID:             id.String(),
		CreatedAt:      at,
		VersionID:      id.String(),
		UpdatedAt:      at,
		Title:          "Amazon Rainforest in Ecuador",
		AltText:        "Amazon Rainforest in Ecuador",
		SourceFileName: "testdata/Ecuador.Rainforest.jpeg",
	}
	imgJPEG, err := service.FetchSourceFile(jpeg.SourceFileName)
	expect.NoError(err, "no error fetching source image")
	expect.NotNil(imgJPEG, "image bytes exist")
	jpeg, err = service.Analyze(jpeg, imgJPEG)
	expect.NoError(err, "no error analyzing image")
	expect.Equal(int64(7097285), jpeg.FileSize, "file size is correct")
	expect.Equal(JPEG, jpeg.MediaType, "media type is correct")
	expect.Equal("90d37d4e2831eef449f3cc0cc8ef4a6d", jpeg.MD5Hash, "MD5 hash is correct")
	expect.Equal("xm3fo2broqae", jpeg.PHash, "pHash is correct")
	expect.Equal(4080, jpeg.Width, "width is correct")
	expect.Equal(3072, jpeg.Height, "height is correct")

	// Analyze a compressed WebP image
	id = tuid.NewID()
	at, _ = id.Time()
	webp := Image{
		ID:             id.String(),
		CreatedAt:      at,
		VersionID:      id.String(),
		UpdatedAt:      at,
		Title:          "Amazon Rainforest in Ecuador",
		AltText:        "Amazon Rainforest in Ecuador",
		SourceFileName: "testdata/Ecuador.Rainforest.webp",
	}
	imgWebP, err := service.FetchSourceFile(webp.SourceFileName)
	expect.NoError(err, "no error fetching source image")
	expect.NotNil(imgWebP, "image bytes exist")
	webp, err = service.Analyze(webp, imgWebP)
	expect.NoError(err, "no error analyzing image")
	expect.Equal(int64(3218748), webp.FileSize, "file size is correct")
	expect.Equal(WebP, webp.MediaType, "media type is correct")
	expect.Equal("ccb6ad951b9af7625a1c9791be0676df", webp.MD5Hash, "MD5 hash is correct")
	expect.Equal("xm3fo2broqae", webp.PHash, "pHash is correct")
	expect.Equal(4080, webp.Width, "width is correct")
	expect.Equal(3072, webp.Height, "height is correct")

	// Compare the perceptual distance between the WebP and JPEG images
	dist, err := PHashDistance(webp.PHash, jpeg.PHash)
	expect.NoError(err, "no error calculating perceptual distance")
	expect.Equal(0, dist, "pHash distance is 0")

	// Compare the perceptual distance between the WebP and PNG images
	dist, err = PHashDistance(webp.PHash, stack64.PHash)
	expect.NoError(err, "no error calculating perceptual distance")
	expect.GreaterOrEqual(dist, 16, "pHash distance is large") // expect 31
}
