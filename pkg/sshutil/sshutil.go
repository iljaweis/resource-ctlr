package sshutil

type Result struct {
	Err      error
	ExitCode int
	Output   string
}

//func FetchPrivateKey(*resourcesv1alpha1.Host, ) (ssh.Signer, error) {
//
//}

// func RunCommand(host *resourcesv1alpha1.Host, command string) {
func RunCommand(host, command string) {

}
