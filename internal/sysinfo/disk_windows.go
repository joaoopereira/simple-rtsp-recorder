package sysinfo

import "golang.org/x/sys/windows"

// Disk returns free and total bytes of the volume containing path.
func Disk(path string) (free, total uint64, err error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	err = windows.GetDiskFreeSpaceEx(p, &free, &total, nil)
	return free, total, err
}
