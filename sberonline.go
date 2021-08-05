package main

import (
    "os"
    "fmt"
    "log"
    "bytes"
    "strconv"
    "strings"
    "net/http"
    "io/ioutil"
    "encoding/json"
)

const appkey = "Thl7ts83WFtt1Ew2fYgGkLbeoD5I3shggfpgPDm5ZukHCpaOx3tEdxrXnnnlhCfR"
const crmurl = "https://crm.domain.ru/crm/api/v1.0"

type Payment struct {
    ClientId int `json:"clientId"`
    MethodId string `json:"methodId"`
    Amount float64 `json:"amount"`
    CurrencyCode string `json:"currencyCode"`
    ApplyToInvoicesAutomatically bool `json:"applyToInvoicesAutomatically"`
    ProviderName string `json:"providerName"`
    ProviderPaymentId string `json:"providerPaymentId"`
}

type Ispclient struct {
    Id int
    UserIdent string
    AddressGpsLat float64
    AddressGpsLon float64
    CompanyName string
    FullAddress string
    AccountBalance float64
    AccountOutstanding float64
    FirstName string
    LastName string
}

func getClientsByuserIdent(userIdent string) []Ispclient {
    httpclient := &http.Client{}
    req, err := http.NewRequest("GET", crmurl+"/clients?userIdent="+userIdent, nil)
    req.Header.Add("Content-Type","application/json")
    req.Header.Add("X-Auth-App-Key", appkey)
    resp, err := httpclient.Do(req)
    if err != nil {
        log.Fatalln(err)
        return nil
    }
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Fatalln(err)
        return nil
    }
    var prettyJSON bytes.Buffer
    error := json.Indent(&prettyJSON, body, "", "\t")
    if error != nil {
        log.Println("JSON parse error: ", error)
        return nil
    }
    var clients []Ispclient
    json.Unmarshal(body,&clients)
    return clients
}

func payAccount(w http.ResponseWriter, r *http.Request) {
    account  := ""
    pay_id   := ""
    amount   := 0.0
    for k, vs := range r.URL.Query() {
        if strings.ToLower(k) == "account"  {account=vs[0]}
        if strings.ToLower(k) == "pay_id"   {pay_id=vs[0]}
        if strings.ToLower(k) == "amount"   {
            amount, _ = strconv.ParseFloat(vs[0], 64)
        }
    }
    if account != "" && pay_id != "" &&  amount > 0 {
        log.Println("Payment account, amount, pay_id:", account, amount, pay_id)
        clients := getClientsByuserIdent(account)
        if len(clients)==0 {
            fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF8\"?>")
            fmt.Fprintf(w, "<response>")
            fmt.Fprintf(w, "<CODE>3</CODE>")
            fmt.Fprintf(w, "<MESSAGE>payment Unsuccessful</MESSAGE>")
            fmt.Fprintf(w, "</response>")
            return
        }
        pay := &Payment{}
        pay.ClientId = clients[0].Id
        pay.Amount = amount
        pay.ProviderPaymentId = pay_id
        pay.CurrencyCode = "RUB"
        pay.ProviderName = "SberOnline"
        pay.ApplyToInvoicesAutomatically = true
        pay.MethodId = "4145b5f5-3bbc-45e3-8fc5-9cda970c62fb"

        jsonStr,_ := json.Marshal(pay)
        log.Println("Payment:", clients)
        log.Println("Payment:", string(jsonStr))

        httpclient := &http.Client{}
        req, _ := http.NewRequest("POST", crmurl+"/payments",bytes.NewBuffer(jsonStr))
        req.Header.Add("Content-Type","application/json")
        req.Header.Add("X-Auth-App-Key", appkey)
        resp, _ := httpclient.Do(req)
        log.Println("Payment_resul:", resp.StatusCode)

        if resp.StatusCode == 201 {
            fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF8\"?>")
            fmt.Fprintf(w, "<response>")
            fmt.Fprintf(w, "<CODE>0</CODE>")
            fmt.Fprintf(w, "<MESSAGE>payment Successful</MESSAGE>")
            fmt.Fprintf(w, "</response>")
            clients := getClientsByuserIdent(account)
            log.Println("Payment:", clients)
        } else {
            fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF8\"?>")
            fmt.Fprintf(w, "<response>")
            fmt.Fprintf(w, "<CODE>1</CODE>")
            fmt.Fprintf(w, "<MESSAGE>payment Unsuccessful</MESSAGE>")
            fmt.Fprintf(w, "</response>")
        }
    }
}

func checkAccount(w http.ResponseWriter, r *http.Request) {
    account:=""
    for k, vs := range r.URL.Query() {
        if strings.ToLower(k) == "account" {account=vs[0]}
    }
    clients := getClientsByuserIdent(account)
    log.Println("Check_account:", account, clients)
    if len(clients) == 1 {
        fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF8\"?>")
        fmt.Fprintf(w, "<response>")
        fmt.Fprintf(w, "<CODE>0</CODE>")
        fmt.Fprintf(w, "<MESSAGE>account exist</MESSAGE>")
        fmt.Fprintf(w, "<FIO>%s %s</FIO>",clients[0].FirstName,clients[0].LastName)
        fmt.Fprintf(w, "<ADDRESS>%s</ADDRESS>",clients[0].FullAddress)
        fmt.Fprintf(w, "<BALANCE>%f</BALANCE>",clients[0].AccountBalance)
        fmt.Fprintf(w, "<REC_SUM>%f</REC_SUM>",clients[0].AccountOutstanding)
        fmt.Fprintf(w, "<INFO></INFO>")
        fmt.Fprintf(w, "</response>")
    } else {
        fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF8\"?>")
        fmt.Fprintf(w, "<response>")
        fmt.Fprintf(w, "<CODE>1</CODE>")
        fmt.Fprintf(w, "<MESSAGE>account %s does not exist</MESSAGE>",account)
        fmt.Fprintf(w, "<INFO></INFO>")
        fmt.Fprintf(w, "</response>")
    }
}

func handler(w http.ResponseWriter, r *http.Request) {
    for k, vs := range r.URL.Query() {
        if strings.ToLower(k) == "action" {
            if strings.ToLower(vs[0]) == "check" {
                checkAccount(w,r)
            }
            if strings.ToLower(vs[0]) == "payment" {
                payAccount(w,r)
            }
        }
    }
}

func main() {
    f, err := os.OpenFile("sberonline.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil {
        log.Fatalf("error opening file: %v", err)
    }
    defer f.Close()
    log.SetOutput(f)
    log.Println("Starting server")

    http.HandleFunc("/", handler)
    http.ListenAndServeTLS(":8080", "server.crt", "server.key", nil)
//    log.Fatal(http.ListenAndServe(":8080", nil))
}
