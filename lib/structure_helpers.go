package lib

import (
	"strings"

	"git.sr.ht/~rjarry/aerc/models"
)

// FindMIMEPart finds the first message part with the provided MIME type.
// FindMIMEPart recurses inside multipart containers.
func FindMIMEPart(mime string, bs *models.BodyStructure, path []int) []int {
	for i, part := range bs.Parts {
		cur := append(path, i+1) //nolint:gocritic // intentional append to different slice
		if part.FullMIMEType() == mime {
			return cur
		}
		if strings.ToLower(part.MIMEType) == "multipart" {
			if path := FindMIMEPart(mime, part, cur); path != nil {
				return path
			}
		}
	}
	return nil
}

func FindPlaintext(bs *models.BodyStructure, path []int) []int {
	return FindMIMEPart("text/plain", bs, path)
}

func FindCalendartext(bs *models.BodyStructure, path []int) []int {
	return FindMIMEPart("text/calendar", bs, path)
}

func FindFirstNonMultipart(bs *models.BodyStructure, path []int) []int {
	for i, part := range bs.Parts {
		cur := append(path, i+1) //nolint:gocritic // intentional append to different slice
		mimetype := strings.ToLower(part.MIMEType)
		if mimetype != "multipart" {
			return cur
		} else if mimetype == "multipart" {
			if path := FindFirstNonMultipart(part, cur); path != nil {
				return path
			}
		}
	}
	return nil
}

func FindAllNonMultipart(bs *models.BodyStructure, path []int, pathlist [][]int) [][]int {
	for i, part := range bs.Parts {
		cur := append(path, i+1) //nolint:gocritic // intentional append to different slice
		mimetype := strings.ToLower(part.MIMEType)
		if mimetype != "multipart" {
			tmp := make([]int, len(cur))
			copy(tmp, cur)
			pathlist = append(pathlist, tmp)
		} else if mimetype == "multipart" {
			if sub := FindAllNonMultipart(part, cur, nil); len(sub) > 0 {
				pathlist = append(pathlist, sub...)
			}
		}
	}
	return pathlist
}

func EqualParts(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
