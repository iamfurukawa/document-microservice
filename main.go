package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gorilla/mux"
	"github.com/klassmann/cpfcnpj"
	"google.golang.org/api/option"
)

func authenticate(token string) bool {
	fmt.Printf("m=authenticate stage=init token=%s\n", token)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "http://localhost:6660/validate", nil)
	req.Header.Add("Authorization", token)
	res, _ := client.Do(req)

	isAuthenticate := res.StatusCode == http.StatusNoContent

	if !isAuthenticate {
		fmt.Printf("m=authenticate stage=end failed to authenticate\n")
		return false
	}

	fmt.Printf("m=authenticate stage=end authorized\n")
	return true

}

func isApproved() bool {
	number := rand.Intn(100)
	fmt.Printf("m=isApproved stage=end number=%d\n", number)
	return number < 90
}

func firebaseRegister(document string, isApproved bool) {
	fmt.Printf("m=firebaseRegister stage=init document=%s isApproved=%t\n", document, isApproved)
	ctx := context.Background()
	opt := option.WithCredentialsFile("./serviceAccount.json")
	client, err := firestore.NewClient(ctx, "estagio-opus", opt)

	if err != nil {
		log.Fatalf("firestore new error:%s\n", err)
	}
	defer client.Close()

	_, err = client.Doc("documents/"+document).Create(ctx, map[string]interface{}{
		"approved": isApproved,
	})

	if err != nil {
		log.Fatalf("firestore Doc Create error:%s\n", err)
	}

	fmt.Printf("m=firebaseRegister stage=end saved on firebase\n")
}

func documentExists(document string) (map[string]interface{}, error) {
	fmt.Printf("m=documentExists stage=init document=%s", document)
	ctx := context.Background()
	opt := option.WithCredentialsFile("./serviceAccount.json")
	client, err := firestore.NewClient(ctx, "estagio-opus", opt)

	if err != nil {
		log.Fatalf("firestore new error:%s\n", err)
	}
	defer client.Close()

	wr, err := client.Doc("documents/" + document).Get(context.Background())

	if err != nil {
		fmt.Printf("m=documentExists stage=end not exists\n")
		return nil, err
	}

	fmt.Printf("m=documentExists stage=end already exists\n")
	return wr.Data(), nil
}

func validateDocument(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	isTokenValid := authenticate(token)

	if !isTokenValid {
		fmt.Printf("m=validateDocument stage=end invalid token: %s\n", token)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	document := mux.Vars(r)["document"]
	fmt.Printf("m=validateDocument stage=init document=%v\n", document)
	cpf := cpfcnpj.NewCPF(document)

	if !cpf.IsValid() {
		fmt.Printf("m=validateDocument stage=end not a valid cpf: %v\n", cpf)
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	data, err := documentExists(document)

	if err == nil {
		approved := data["approved"].(bool)
		fmt.Printf("m=validateDocument stage=end approved: %t\n", approved)
		if approved {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	wasApproved := isApproved()

	firebaseRegister(document, wasApproved)

	fmt.Printf("m=validateDocument stage=end is valid")
	if !wasApproved {
		fmt.Printf("m=validateDocument stage=end but not approved")
		w.WriteHeader(http.StatusBadRequest)
	}

	fmt.Printf("m=validateDocument stage=end approved")
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/document/{document}", validateDocument)
	fmt.Printf("Server starting...\n")
	log.Fatal(http.ListenAndServe(":6661", router))
}
