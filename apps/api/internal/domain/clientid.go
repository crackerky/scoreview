package domain

// IsClientID は value が端末の作る操作の ID(UUID の 8-4-4-4-12 の 16 進表記)かどうか。
// 通信が切れている間に記録した操作を後から送り直しても二重に登録しないための ID で、
// 得点経過を api の id を知らずに指すためにも使う。
func IsClientID(value string) bool {
	const length = 36
	if len(value) != length {
		return false
	}
	for i, char := range value {
		switch i {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			if !isHexDigit(char) {
				return false
			}
		}
	}
	return true
}

func isHexDigit(char rune) bool {
	return (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')
}

// CheckClientID は nil か UUID の形であることを確かめる。
func CheckClientID(field string, value *string) error {
	if value != nil && !IsClientID(*value) {
		return Invalid(field, "must be a UUID")
	}
	return nil
}
