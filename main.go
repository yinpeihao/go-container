package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

//docker 		 run image <cmd> <params>
//go run main.go run image
func main() {
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("bad command")
	}
}

func run() {
	fmt.Printf("Running %v as %d\n", os.Args[2:], os.Getpid())
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		//利用bit位，使用|操作符来指定多个clone flag
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func pidControl(maxPids int) {
	pidCg := "/sys/fs/cgroup/pids"
	groupPath := filepath.Join(pidCg, "/gocg")
	//创建gocg组
	err := os.Mkdir(groupPath, 0775)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}
	//最多的pid数量
	must(ioutil.WriteFile(filepath.Join(groupPath, "pids.max"), []byte(strconv.Itoa(maxPids)), 0700))
	//将当前进程加入到gocg组
	must(ioutil.WriteFile(filepath.Join(groupPath, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}

func cpuControl(core float64) {
	pidCg := "/sys/fs/cgroup/cpu"
	groupPath := filepath.Join(pidCg, "/gocg")
	//创建gocg组
	err := os.Mkdir(groupPath, 0775)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}
	//10ms
	cfs := float64(10000)
	//cpu配额
	must(ioutil.WriteFile(filepath.Join(groupPath, "cpu.cfs_quota_us"), []byte(strconv.Itoa(int(cfs*core))), 0700))
	//时间周期
	must(ioutil.WriteFile(filepath.Join(groupPath, "cpu.cfs_period_us"), []byte(strconv.Itoa(int(cfs))), 0700))
	//将当前进程加入到gocg组
	must(ioutil.WriteFile(filepath.Join(groupPath, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func child() {
	//限制20个进程
	pidControl(20)
	//限制使用的cpu为0.5核
	cpuControl(0.5)

	fmt.Printf("Running %v as %d\n", os.Args[2:], os.Getpid())
	syscall.Chroot("/home/yinpeihao/go/src/github.com/yinpeihao/implement-container/apline")
	syscall.Chdir("/")
	syscall.Mount("proc", "proc", "proc", 0, "")
	syscall.Sethostname([]byte("container"))

	defer syscall.Unmount("proc", 0)
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}
