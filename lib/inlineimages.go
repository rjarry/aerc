package lib

import (
	"bytes"
	"encoding/base64"
	"io"
	"maps"
	"strings"
	"sync"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"golang.org/x/net/html"
)

// InlineHTMLImages transforms an HTML email by replacing <img> tags with cid:
// URLs with base64-encoded data: URLs. This allows HTML emails with embedded
// images to be viewed in browsers that support data: URLs, including w3m when
// used with the -sixel option and img2sixel installed.
//
// This function uses callbacks and will call the provided callback
// asynchronously once all images have been fetched and inlined.
func InlineHTMLImages(
	htmlReader io.Reader,
	msg MessageView,
	callback func(io.Reader),
) {
	// Read the HTML content
	htmlBytes, err := io.ReadAll(htmlReader)
	if err != nil {
		log.Errorf("Failed to read HTML: %v", err)
		callback(bytes.NewReader(htmlBytes))
		return
	}

	// Parse the HTML
	doc, err := html.Parse(bytes.NewReader(htmlBytes))
	if err != nil {
		log.Errorf("Failed to parse HTML: %v", err)
		callback(bytes.NewReader(htmlBytes))
		return
	}

	// Build a map of Content-ID to part index
	cidMap := buildContentIDMap(msg.BodyStructure(), []int{})

	// If no Content-IDs found, return the original HTML
	if len(cidMap) == 0 {
		log.Tracef("No Content-IDs found in message, skipping image inlining")
		callback(bytes.NewReader(htmlBytes))
		return
	}

	// Find all cid: references in img tags
	cidReferences := make(map[string]bool)
	var findCids func(*html.Node)
	findCids = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			for _, attr := range n.Attr {
				if attr.Key == "src" && strings.HasPrefix(attr.Val, "cid:") {
					cid := strings.Trim(strings.TrimPrefix(attr.Val, "cid:"), "<>")
					if _, ok := cidMap[cid]; ok {
						cidReferences[cid] = true
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findCids(c)
		}
	}
	findCids(doc)

	if len(cidReferences) == 0 {
		log.Tracef("No cid: references found in HTML, returning original")
		callback(bytes.NewReader(htmlBytes))
		return
	}

	log.Tracef("Found %d cid: references to inline", len(cidReferences))

	// Fetch all images asynchronously and encode as data: URLs
	var wg sync.WaitGroup
	var mu sync.Mutex
	imageURLs := make(map[string]string)

	for cid := range cidReferences {
		info := cidMap[cid]
		wg.Add(1)

		msg.FetchBodyPart(info.index, func(reader io.Reader) {
			defer wg.Done()

			data, err := io.ReadAll(reader)
			if err != nil {
				log.Errorf("Failed to read image part for CID %s: %v", cid, err)
				return
			}

			// Encode as base64
			encoded := base64.StdEncoding.EncodeToString(data)

			// Create data: URL
			dataURL := "data:" + info.mimeType + ";base64," + encoded

			mu.Lock()
			imageURLs[cid] = dataURL
			mu.Unlock()

			log.Tracef("Encoded image with Content-ID %s as data: URL (%d bytes)", cid, len(data))
		})
	}

	// Wait for all images to be fetched, then transform HTML
	go func() {
		defer log.PanicHandler()
		wg.Wait()

		// Replace all cid: references with data: URLs
		modified := false
		var processNode func(*html.Node)
		processNode = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "img" {
				for i, attr := range n.Attr {
					if attr.Key == "src" && strings.HasPrefix(attr.Val, "cid:") {
						cid := strings.Trim(strings.TrimPrefix(attr.Val, "cid:"), "<>")
						if dataURL, ok := imageURLs[cid]; ok {
							n.Attr[i].Val = dataURL
							modified = true
							log.Tracef("Replaced cid:%s with data: URL", cid)
						}
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				processNode(c)
			}
		}
		processNode(doc)

		if !modified {
			log.Warnf("No images were inlined despite finding CID references")
			callback(bytes.NewReader(htmlBytes))
			return
		}

		// Render the modified HTML back to bytes
		var buf bytes.Buffer
		err = html.Render(&buf, doc)
		if err != nil {
			log.Errorf("Failed to render HTML: %v", err)
			callback(bytes.NewReader(htmlBytes))
			return
		}

		callback(&buf)
	}()
}

// partInfo stores the index and MIME type of a part
type partInfo struct {
	index    []int
	mimeType string
}

// buildContentIDMap recursively builds a map from Content-ID to part information
func buildContentIDMap(bs *models.BodyStructure, index []int) map[string]partInfo {
	result := make(map[string]partInfo)

	if bs == nil {
		log.Tracef("buildContentIDMap: body structure is nil")
		return result
	}

	// Log the current part being examined
	log.Tracef("buildContentIDMap: examining part index=%v mime=%s contentid=%q numParts=%d",
		index, bs.FullMIMEType(), bs.ContentID, len(bs.Parts))

	// Add this part if it has a Content-ID
	if bs.ContentID != "" {
		result[strings.Trim(bs.ContentID, "<>")] = partInfo{
			index:    append([]int{}, index...),
			mimeType: bs.FullMIMEType(),
		}
		log.Tracef("Found Content-ID: %s at index %v (%s)",
			bs.ContentID, index, bs.FullMIMEType())
	}

	// Recursively process child parts
	for i, part := range bs.Parts {
		childIndex := append(append([]int{}, index...), i+1)
		maps.Copy(result, buildContentIDMap(part, childIndex))
	}

	return result
}
