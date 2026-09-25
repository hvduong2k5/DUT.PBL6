package usecase

import (
	"strings"
	"unicode"
)

type AdministrativeCandidate struct {
	Code  string
	Name  string
	Level string
}

type MatchResult struct {
	Code  string
	Name  string
	Score float64
}

// StripVietnameseTones converts accented Vietnamese characters to their ASCII base.
func StripVietnameseTones(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'á', 'à', 'ả', 'ã', 'ạ', 'â', 'ấ', 'ầ', 'ẩ', 'ẫ', 'ậ', 'ă', 'ắ', 'ằ', 'ẳ', 'ẵ', 'ặ':
			sb.WriteRune('a')
		case 'é', 'è', 'ẻ', 'ẽ', 'ẹ', 'ê', 'ế', 'ề', 'ể', 'ễ', 'ệ':
			sb.WriteRune('e')
		case 'i', 'í', 'ì', 'ỉ', 'ĩ', 'ị':
			sb.WriteRune('i')
		case 'ó', 'ò', 'ỏ', 'õ', 'ọ', 'ô', 'ố', 'ồ', 'ổ', 'ỗ', 'ộ', 'ơ', 'ớ', 'ờ', 'ở', 'ỡ', 'ợ':
			sb.WriteRune('o')
		case 'ú', 'ù', 'ủ', 'ũ', 'ụ', 'ư', 'ứ', 'ừ', 'ử', 'ữ', 'ự':
			sb.WriteRune('u')
		case 'ý', 'ỳ', 'ỷ', 'ỹ', 'ỵ':
			sb.WriteRune('y')
		case 'đ':
			sb.WriteRune('d')
		default:
			if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
				sb.WriteRune(r)
			}
		}
	}
	return sb.String()
}

// CleanAdministrativeStopwords removes administrative prefixes and suffixes.
func CleanAdministrativeStopwords(s string) string {
	lowered := strings.ToLower(s)

	// Replace punctuation with spaces
	replacer := strings.NewReplacer(".", " ", ",", " ", "-", " ", "/", " ")
	cleaned := replacer.Replace(lowered)
	cleaned = StripVietnameseTones(cleaned)

	stopwords := []string{
		"thua thien hue", "tp hue", "thanh pho hue", "huong thuy", "tx", "tp", "tt",
		"phuong", "xa", "thi tran", "thi xa", "huyen", "tinh", "thanh pho",
		"p ", "x ", "cho dong ba", "hue",
	}

	for _, stop := range stopwords {
		cleaned = strings.ReplaceAll(cleaned, stop, " ")
	}

	// Normalize spaces
	words := strings.Fields(cleaned)
	return strings.Join(words, " ")
}

// LevenshteinDistance computes Levenshtein distance between two rune slices with Vietnamese phonetic weighting.
func LevenshteinDistance(s1, s2 []rune) float64 {
	len1, len2 := len(s1), len(s2)
	matrix := make([][]float64, len1+1)
	for i := range matrix {
		matrix[i] = make([]float64, len2+1)
		matrix[i][0] = float64(i)
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = float64(j)
	}

	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0.0
			r1, r2 := s1[i-1], s2[j-1]
			if r1 != r2 {
				// Vietnamese phonetic equivalence: 'i' and 'y' share identical pronunciation
				if (r1 == 'i' && r2 == 'y') || (r1 == 'y' && r2 == 'i') {
					cost = 0.5
				} else {
					cost = 1.0
				}
			}
			matrix[i][j] = min3Float(
				matrix[i-1][j]+1.0,    // Deletion
				matrix[i][j-1]+1.0,    // Insertion
				matrix[i-1][j-1]+cost, // Substitution
			)
		}
	}
	return matrix[len1][len2]
}

func min3Float(a, b, c float64) float64 {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

// CalculateRefinedSimilarity evaluates similarity score between [0.0, 1.0].
func CalculateRefinedSimilarity(userInput, candidateName string) float64 {
	uClean := CleanAdministrativeStopwords(userInput)
	cClean := CleanAdministrativeStopwords(candidateName)

	if uClean == cClean {
		return 1.0
	}

	// Also check character sequence without spaces (concatenated handling)
	uCompact := strings.ReplaceAll(uClean, " ", "")
	cCompact := strings.ReplaceAll(cClean, " ", "")
	if uCompact == cCompact && len(cCompact) > 0 {
		return 0.95
	}

	// Substring bonus
	if strings.Contains(uClean, cClean) || strings.Contains(cClean, uClean) {
		diff := len(uClean) - len(cClean)
		if diff < 0 {
			diff = -diff
		}
		penalty := float64(diff) * 0.02
		score := 0.95 - penalty
		if score > 0.88 {
			return score
		}
	}

	r1, r2 := []rune(uClean), []rune(cClean)
	dist := LevenshteinDistance(r1, r2)
	maxLen := len(r1)
	if len(r2) > maxLen {
		maxLen = len(r2)
	}
	if maxLen == 0 {
		return 1.0
	}

	score := 1.0 - float64(dist)/float64(maxLen)
	if score < 0 {
		return 0.0
	}
	return score
}

type AddressFuzzyMatcher struct{}

func NewAddressFuzzyMatcher() *AddressFuzzyMatcher {
	return &AddressFuzzyMatcher{}
}

// RefineCandidates ranks pre-filtered DB candidates and returns the best matching candidate.
func (m *AddressFuzzyMatcher) RefineCandidates(userInput string, candidates []AdministrativeCandidate) *MatchResult {
	if len(candidates) == 0 {
		return nil
	}

	var best MatchResult
	for _, c := range candidates {
		score := CalculateRefinedSimilarity(userInput, c.Name)
		if score > best.Score {
			best = MatchResult{
				Code:  c.Code,
				Name:  c.Name,
				Score: score,
			}
		}
	}

	return &best
}
