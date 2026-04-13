//go:build !darwin && !windows

package wechat

type otherPlatform struct{}

func newPlatform() platform { return &otherPlatform{} }

func (m *otherPlatform) GetDefaultPaths(log ...LogFunc) []string {
	return []string{}
}
