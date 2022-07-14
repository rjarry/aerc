package lib

import (
	"strings"

	"git.sr.ht/~rjarry/aerc/models"
)

func FindPlaintext(bs *models.BodyStructure, path []int) []int {
	for i, part := range bs.Parts {
		cur := append(path, i+1)
		if strings.ToLower(part.MIMEType) == "text" &&
			strings.ToLower(part.MIMESubType) == "plain" {
			return cur
		}
		if strings.ToLower(part.MIMEType) == "multipart" {
			if path := FindPlaintext(part, cur); path != nil {
				return path
			}
		}
	}
	return nil
}

func FindCalendartext(bs *models.BodyStructure, path []int) []int {
	for i, part := range bs.Parts {
		cur := append(path, i+1)
		if strings.ToLower(part.MIMEType) == "text" &&
			strings.ToLower(part.MIMESubType) == "calendar" {
			return cur
		}
		if strings.ToLower(part.MIMEType) == "multipart" {
			if path := FindCalendartext(part, cur); path != nil {
				return path
			}
		}
	}
	return nil
}

func FindFirstNonMultipart(bs *models.BodyStructure, path []int) []int {
	for i, part := range bs.Parts {
		cur := append(path, i+1)
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
		cur := append(path, i+1)
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
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
