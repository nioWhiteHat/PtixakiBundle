package main

import (
	p "Ptixiaki/Backend/Api"
	mar "Ptixiaki/Backend/Api/MarketAnalysis"
	u "Ptixiaki/Backend/Api/UserActions"
	auth "Ptixiaki/Backend/Auth"
	geo "Ptixiaki/Backend/ExternalServices/Geo"
	db "Ptixiaki/Database"
	tr "Ptixiaki/Worker/Pipelines"
	explain "Ptixiaki/Worker/explainability"
	c "Ptixiaki/Worker/testing"
	t "Ptixiaki/Worker/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yalue/onnxruntime_go"
)

var TreePaths = []string{

	"Worker/Models/TreeOutputs/Phase2_residence_properties_sell_Tree.onnx",
	"Worker/Models/TreeOutputs/Phase2_residence_properties_rent_Tree.onnx",
}

type application struct {
	AreaCatalog  mar.AreaCatalog
	Mu           sync.RWMutex
	TrainingMu   sync.Mutex
	Calculator   c.Calculator
	SellTreeAttr *t.ModelPredictor
	RentTreeAttr *t.ModelPredictor
	SellTree     *t.ModelPredictor
	RentTree     *t.ModelPredictor
}

func main() {

	app := &application{}
	err := app.WarmUpApplication()
	if err != nil {
		log.Fatalf("Failed to warm up application: %v", err)
	}

	http.HandleFunc("/predict/sell", app.PredictSellHandler)
	//http.HandleFunc("/predict/rent", app.PredictRentHandler)
	http.HandleFunc("/train", app.TrainTreesHandler)
	http.HandleFunc("/register/SignUp", app.SignUpHandler)
	http.HandleFunc("/register/SignIn", app.SignInHandler)
	http.HandleFunc("/User/AddProperty", auth.Authenticate(app.AddPropertyHandler))
	http.HandleFunc("/User/ViewProperty", auth.Authenticate(app.ViewPropertyHandler))
	http.HandleFunc("/User/BrowseProperties", auth.Authenticate(app.BrowsePropertiesHandler))
	http.HandleFunc("/User/BrowseProperty", auth.Authenticate(app.BrowsePropertyHandler))
	http.HandleFunc("/User/Portfolio", auth.Authenticate(app.PortfolioHandler))
	http.HandleFunc("/User/DeleteProperty", auth.Authenticate(app.DeletePropertyHandler))
	http.HandleFunc("/test", app.test)
	http.HandleFunc("/analysis/autocomplete", auth.Authenticate(app.AutocompleteHandler))
	http.HandleFunc("/analysis/CreateAnalysisForArea", auth.Authenticate(app.CreateAnalysisForAreaHandler))
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}

}

type AreaAnalysisRequest struct {
	BoundingBox []string `json:"bounding_box"`
}

type AnalysisResponse struct {
	Points      []mar.DataPoint `json:"points"`
	Clusters    []mar.Cluster   `json:"clusters"`
	SalesReport *mar.MLResponse `json:"sales_report"`
	RentReport  *mar.MLResponse `json:"rent_report"`
	Warnings    []string        `json:"warnings,omitempty"`
}

func (app *application) CreateAnalysisForAreaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST", http.StatusMethodNotAllowed)
		return
	}
	var bbox mar.BoundingBox

	err := json.NewDecoder(r.Body).Decode(&bbox)
	if err != nil {
		analysisError(w, "invalid_area", http.StatusBadRequest)
		return
	}
	if bbox.MinLat >= bbox.MaxLat || bbox.MinLon >= bbox.MaxLon || bbox.MinLat < -90 || bbox.MaxLat > 90 || bbox.MinLon < -180 || bbox.MaxLon > 180 {
		analysisError(w, "invalid_area", http.StatusBadRequest)
		return
	}

	ap := &mar.AnalysisPipeline{
		Db:  app.Calculator.Db,
		Ctx: r.Context(),
	}

	points, clusters, salesML, rentML, err := ap.RunPipeline(bbox)
	if err != nil {
		log.Printf("[ANALYSIS] Failed: %v", err)
		analysisError(w, "analysis_data_unavailable", http.StatusInternalServerError)
		return
	}

	response := AnalysisResponse{
		Points:      points,
		Clusters:    clusters,
		SalesReport: salesML,
		RentReport:  rentML,
		Warnings:    ap.Warnings,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *application) AutocompleteHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	var results any
	var err error
	if r.URL.Query().Get("kind") == "address" {
		results, err = geo.Geocoder.Search(ctx, query)
	} else {
		results, err = a.AreaCatalog.Search(ctx, a.Calculator.Db, query)
	}
	if err != nil {
		log.Printf("[AREA SEARCH] Search failed")
		analysisError(w, "area_search_unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func analysisError(w http.ResponseWriter, code string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code}})
}

type ViewPropertyRequest struct {
	PropertyUid string `json:"PropertyUid"`
}

func (a *application) ViewPropertyHandler(w http.ResponseWriter, r *http.Request) {
	u.ServePropertyDetails(w, r, a.Calculator.Db)
}

func (a *application) BrowsePropertiesHandler(w http.ResponseWriter, r *http.Request) {
	u.ServeBrowseProperties(w, r, a.Calculator.Db)
}

func (a *application) BrowsePropertyHandler(w http.ResponseWriter, r *http.Request) {
	u.ServeBrowsePropertyDetails(w, r, a.Calculator.Db)
}

func (a *application) PortfolioHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := &u.User{Uid: r.Context().Value("user_uid").(string)}
	portfolio, err := user.GetPortfolio(a.Calculator.Db, r.Context())
	if err != nil {
		http.Error(w, "failed to load portfolio", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(portfolio); err != nil {
		log.Printf("failed to encode portfolio response: %v", err)
	}
}

func (a *application) DeletePropertyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	propertyUID := strings.TrimSpace(r.URL.Query().Get("property_uid"))
	if propertyUID == "" {
		http.Error(w, "property_uid is required", http.StatusBadRequest)
		return
	}

	user := &u.User{Uid: r.Context().Value("user_uid").(string)}
	removed, err := user.RemovePropertyFromPortfolio(a.Calculator.Db, r.Context(), propertyUID)
	if err != nil {
		http.Error(w, "failed to remove property from portfolio", http.StatusInternalServerError)
		return
	}
	if !removed {
		http.Error(w, "property not found in portfolio", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Property removed from portfolio successfully"}); err != nil {
		log.Printf("failed to encode delete-property response: %v", err)
	}
}

func (a *application) test(w http.ResponseWriter, r *http.Request) {

	err := tr.InsertToDb(a.Calculator.Ctx, a.Calculator.Db, "Worker/Models/GeoPremiums/GeoPrem_residence_properties_sell.json", "Worker/Models/JsonOutputs/Attr_residence_properties_sell_PricePredictions.json", "Worker/Models/JsonOutputs/Phase2_residence_properties_sell_PricePredictions.json", "sell")
	if err != nil {
		log.Fatal(err)
	}
}
func (a *application) AddPropertyHandler(w http.ResponseWriter, r *http.Request) {
	u.ServeAddProperty(w, r, func(ctx context.Context, userUID string, prop p.PropertyFeatures, transaction string) (string, error) {
		a.Mu.RLock()
		defer a.Mu.RUnlock()
		var attrTree, predTree *t.ModelPredictor
		if transaction == "sell" {
			attrTree, predTree = a.SellTreeAttr, a.SellTree
		} else {
			attrTree, predTree = a.RentTreeAttr, a.RentTree
		}
		user := u.User{Uid: userUID}
		return user.AddProperty(a.Calculator.Db, ctx, prop, attrTree, predTree, transaction)
	})
}

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *application) SignInHandler(w http.ResponseWriter, r *http.Request) {
	var req SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authInst := &auth.Auth{
		Db:  a.Calculator.Db,
		Ctx: r.Context(),
	}

	uid, admin, err := authInst.SignIn(req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	token, err := authInst.CreateToken(uid, admin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User signed in successfully!"))
}

type SignUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Admin    bool   `json:"admin"`
}

func (a *application) SignUpHandler(w http.ResponseWriter, r *http.Request) {
	var req SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	validPassword := regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(req.Password)
	if !validPassword {
		http.Error(w, "password must contain only english letters and numbers", http.StatusBadRequest)
		return
	}

	authInst := &auth.Auth{
		Db:  a.Calculator.Db,
		Ctx: r.Context(),
	}

	uid, _, err := authInst.CreateUser(req.Email, req.Password, req.Admin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := authInst.CreateToken(uid, req.Admin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User created successfully!"))
}
func (a *application) TrainTreesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("starting")
	err := a.trainResidenceTrees(r.Context())
	if err != nil {
		log.Printf("training/reload failed: %v", err)
		http.Error(w, "training or model reload failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Training completed successfully!"))

}
func (a *application) TrainTrees() error {
	return a.trainResidenceTrees(a.Calculator.Ctx)
}

func (a *application) trainResidenceTrees(ctx context.Context) error {
	a.TrainingMu.Lock()
	defer a.TrainingMu.Unlock()

	trainer := tr.InitTrainingPipeline(a.Calculator.Db, ctx)
	err := trainer.TrainJob("residence", "properties", "sell")
	if err != nil {
		return fmt.Errorf("sell training failed: %w", err)
	}
	err = trainer.TrainJob("residence", "properties", "rent")
	if err != nil {
		return fmt.Errorf("rent training failed: %w", err)
	}

	sellTree, rentTree, err := loadTreePair()
	if err != nil {
		return fmt.Errorf("validate newly trained ONNX models: %w", err)
	}
	expectedHashes, err := explain.LoadLocalModelHashes(
		filepath.Dir(TreePaths[0]),
		[]string{"sell", "rent"},
	)
	if err != nil {
		destroyPredictor(sellTree)
		destroyPredictor(rentTree)
		return fmt.Errorf("validate newly trained native model publications: %w", err)
	}

	reloadCtx, cancelReload := context.WithTimeout(ctx, explain.HTTPTimeout)
	reloadResponse, err := explain.NewClient().ReloadModels(
		reloadCtx,
		[]string{"sell", "rent"},
		expectedHashes,
	)
	cancelReload()
	if err != nil {
		destroyPredictor(sellTree)
		destroyPredictor(rentTree)
		return fmt.Errorf("reload Python explanation models: %w", err)
	}

	a.publishTrees(sellTree, rentTree)
	log.Printf(
		"Go and Python sell/rent models reloaded successfully (sell=%s rent=%s)",
		reloadResponse.Models[0].ModelSHA256,
		reloadResponse.Models[1].ModelSHA256,
	)
	return nil
}

func (a *application) PredictSellHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("uid")
	a.Mu.RLock()
	prediction, err := c.PredictPricePipeline(uid, a.SellTreeAttr, a.SellTree, a.Calculator)
	a.Mu.RUnlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Predicted Sell Price: %f", prediction.Price)))

}
func (a *application) WarmUpApplication() error {
	con, err := db.InitDB1()
	if err != nil {
		return err
	}
	Calculator := c.Calculator{
		Db:  con,
		Ctx: context.Background(),
	}
	a.Calculator = Calculator
	err = a.WarmUpTrees()
	if err != nil {
		return err
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func (a *application) WarmUpTrees() error {

	onnxruntime_go.SetSharedLibraryPath("onnxengine.dll")

	err := onnxruntime_go.InitializeEnvironment()
	if err != nil {
		return err
	}

	flag := false
	if fileExists(TreePaths[0]) && fileExists(TreePaths[1]) {
		flag = true
	}
	if !flag {
		err := a.TrainTrees()
		if err != nil {
			return fmt.Errorf("failed to train trees: %v", err)
		}
		return nil
	}
	return a.LoadOrReloadTrees()
}

func loadTree(modelPath string, featureCount int) (*t.ModelPredictor, error) {
	inputData := make([]float32, featureCount)
	inputShape := onnxruntime_go.NewShape(1, int64(featureCount))
	inputTensor, err := onnxruntime_go.NewTensor(inputShape, inputData)
	if err != nil {
		return nil, err
	}

	outputData := make([]float32, 1)
	outputShape := onnxruntime_go.NewShape(1, 1)
	outputTensor, err := onnxruntime_go.NewTensor(outputShape, outputData)
	if err != nil {
		inputTensor.Destroy()
		return nil, err
	}

	session, err := onnxruntime_go.NewAdvancedSession(
		modelPath,
		[]string{"input"},
		[]string{"variable"},
		[]onnxruntime_go.Value{inputTensor},
		[]onnxruntime_go.Value{outputTensor},
		nil,
	)
	if err != nil {
		inputTensor.Destroy()
		outputTensor.Destroy()
		return nil, err
	}

	predictor := &t.ModelPredictor{
		Session:      session,
		InputTensor:  inputTensor,
		OutputTensor: outputTensor,
		InputData:    inputData,
		OutputData:   outputData,
	}
	if _, err := c.PredictPrice(make([]float32, featureCount), predictor); err != nil {
		destroyPredictor(predictor)
		return nil, fmt.Errorf("model validation run failed: %w", err)
	}
	return predictor, nil
}

func (a *application) LoadOrReloadTrees() error {
	sellTree, rentTree, err := loadTreePair()
	if err != nil {
		return err
	}
	a.publishTrees(sellTree, rentTree)
	log.Println("Trees loaded successfully")
	return nil
}

func loadTreePair() (*t.ModelPredictor, *t.ModelPredictor, error) {
	sellTree, err := loadTree(TreePaths[0], explain.FeatureCount)
	if err != nil {
		return nil, nil, fmt.Errorf("load sell tree: %w", err)
	}
	rentTree, err := loadTree(TreePaths[1], explain.FeatureCount)
	if err != nil {
		destroyPredictor(sellTree)
		return nil, nil, fmt.Errorf("load rent tree: %w", err)
	}
	return sellTree, rentTree, nil
}

func (a *application) publishTrees(sellTree, rentTree *t.ModelPredictor) {
	a.Mu.Lock()
	oldSell := a.SellTree
	oldRent := a.RentTree
	a.SellTree = sellTree
	a.RentTree = rentTree

	// Every prediction path holds Mu.RLock while it owns a model pointer. Taking
	// the write lock above therefore waits for old sessions to become idle and
	// prevents a new request from observing them before destruction.
	destroyPredictor(oldSell)
	destroyPredictor(oldRent)
	a.Mu.Unlock()
}

func destroyPredictor(predictor *t.ModelPredictor) {
	if predictor == nil {
		return
	}
	predictor.Mu.Lock()
	defer predictor.Mu.Unlock()
	if predictor.Session != nil {
		predictor.Session.Destroy()
		predictor.Session = nil
	}
	if predictor.InputTensor != nil {
		predictor.InputTensor.Destroy()
		predictor.InputTensor = nil
	}
	if predictor.OutputTensor != nil {
		predictor.OutputTensor.Destroy()
		predictor.OutputTensor = nil
	}
}
