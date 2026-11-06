package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractStreetName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"29 Mai Lão Bạng", "Mai Lão Bạng"},
		{"Số 131/4B Lê Lợi", "Lê Lợi"},
		{"Kiệt 42 Nguyễn Huệ", "Nguyễn Huệ"},
		{"Nhà số 5, Tràng Tiền", "Tràng Tiền"},
		{"Ngõ 12 Đinh Tiên Hoàng", "Đinh Tiên Hoàng"},
		{"Hẻm 45 Hai Bà Trưng", "Hai Bà Trưng"},
		{"2 Tháng 9", "2 Tháng 9"}, // Con đường có số trong tên
		{"Đường Chi Lăng", "Đường Chi Lăng"},
		{"123 Nguyễn Văn Linh", "Nguyễn Văn Linh"},
	}

	for _, tc := range testCases {
		actual := ExtractStreetName(tc.input)
		assert.Equal(t, tc.expected, actual, "Failed extracting street from %s", tc.input)
	}
}
