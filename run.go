package main

import (
  "syscall"
  "os"
  "fmt"
  "log"

  "net/mail"
  "encoding/base64"
  "net/smtp"
  "sync"
  "time"
)

type MailerStruct struct {
  mb_available uint64
  mb_at_least uint64
  send_mail_to string
  smtp_credentials string

  wait_group sync.WaitGroup
}

func SendEmail(wrapper *MailerStruct) {

  defer wrapper.wait_group.Done()

  fmt.Println("Sending notifications to ", wrapper.send_mail_to, "...")
  
  smtpServer := "smtp.postmarkapp.com"
  auth := smtp.PlainAuth(
    "",
    wrapper.smtp_credentials,
    wrapper.smtp_credentials,
    smtpServer,
  )
 
  from := mail.Address{"FROM_NAME", "from@example.com"}
  to := mail.Address{"SystemUser", wrapper.send_mail_to}
  title := "Alert: System Status Check."

  body := fmt.Sprintf("<h3>Hi there,</h3> <p> The ScoutOnDemand server <b>hard drive is almost full.</b> Only <b>%d MBytes</b> left on device. (We should have at least %d Mb) </p> <p>Please contact system administrator to fix this issue.</p> <br/> <hr /> ScoutOnDemand system Monitor", wrapper.mb_available, wrapper.mb_at_least)

  header := make(map[string]string)
  header["From"] = from.String()
  header["To"] = to.String()
  header["Subject"] = title
  header["MIME-Version"] = "1.0"
  header["Content-Type"] = "text/html; charset=\"utf-8\""
  header["Content-Transfer-Encoding"] = "base64"
 
  message := ""
  for k, v := range header {
    message += fmt.Sprintf("%s: %s\r\n", k, v)
  }
  message += "\r\n" + base64.StdEncoding.EncodeToString([]byte(body))
 
  // Connect to the server, authenticate, set the sender and recipient,
  // and send the email all in one step.
  err := smtp.SendMail(
    smtpServer + ":2525",
    auth,
    from.Address,
    []string{to.Address},
    []byte(message),
  )
  if err != nil {
    log.Fatal(err)
  }
}

func checkForDiskUsage(to_wait int) int {
  const lower_limit_hdd_usage uint64 =  512 // mb
  var stat syscall.Statfs_t
  next_test_in := to_wait

  wd, err := os.Getwd()

  if err != nil {
    log.Fatal(err)
  }

  syscall.Statfs(wd, &stat)

  var left_megabytes uint64

  left_megabytes = stat.Bavail * uint64(stat.Bsize) / 1024 / 1024

  mailer_wrapper := &MailerStruct{
    mb_available: left_megabytes,
    mb_at_least: lower_limit_hdd_usage,
    send_mail_to: os.Args[1],
    smtp_credentials: os.Args[2]}

  if left_megabytes < lower_limit_hdd_usage {
    fmt.Println("Only ", left_megabytes, "Mb left on HDD. We should have at least ", lower_limit_hdd_usage, " Mb.")
    mailer_wrapper.wait_group.Add(1)

    if to_wait >= 48 {
      next_test_in = 2
    } else {
      next_test_in = to_wait * 2
    }

    go SendEmail(mailer_wrapper)
  } else {
    next_test_in = 1
  }

  mailer_wrapper.wait_group.Wait()

  return next_test_in
}

func main() {
  to_wait := 1 // in hours
  for {
    if to_wait == 1 {
      fmt.Println("We have some space on HDD. Monitoring...")
    } else {
      fmt.Println("Next notification will be sent in ", to_wait, " hours.")
    }
    to_wait = checkForDiskUsage(to_wait)
    time.Sleep(time.Duration(to_wait) * time.Hour)
  }
}
