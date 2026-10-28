# Πίνακας ορολογίας
@table Βασικοί ελληνικοί και αγγλικοί όροι
| Ελληνικός όρος | Αγγλικός όρος | Χρήση στο έργο |
| --- | --- | --- |
| Ζητούμενη τιμή | Asking price | Τιμή αγγελίας, όχι κατ’ ανάγκη ολοκληρωμένης συναλλαγής |
| Εκτίμηση | Prediction / estimate | Έξοδος μοντέλου σε €/m² |
| Χαρακτηριστικό | Feature | Μία από τις εισόδους του μοντέλου |
| Ετικέτα / στόχος | Label / target | AskingPricePerSqm κατά την εκπαίδευση |
| Κατάλοιπο | Residual | Απόκλιση asking από prediction |
| Τιμή αναφοράς | Baseline | Σταθερός όρος της αποσύνθεσης SHAP |
| Συνεισφορά | Contribution | Προσημασμένη επίδραση χαρακτηριστικού στη μοντελοποιημένη τιμή |
| Σημασία χαρακτηριστικού | Feature importance | Συγκεντρωτικό ή τοπικό μέτρο συμμετοχής |
| Αλληλεπίδραση | Interaction | Συνδυαστική συμμετοχή δύο χαρακτηριστικών |
| Χωρική προσαύξηση | GeoPremium | Εσωτερική ονομασία μέσου Attr καταλοίπου γειτόνων |
| Τυποποιημένη απόκλιση | Z-score | Απόσταση από μέσο σε μονάδες τυπικής απόκλισης |
| Ενίσχυση κλίσης | Gradient boosting | Διαδοχική προσθήκη δέντρων για μείωση απώλειας |
| Ομαδοποίηση | Clustering | Περιγραφική δημιουργία ομάδων ομοιότητας |
| Επικύρωση | Validation | Έλεγχος σε διαχωρισμένο σύνολο, με τους δηλωμένους περιορισμούς |
| Διαρροή πληροφορίας | Data leakage | Χρήση μη επιτρεπτής πληροφορίας κατά αξιολόγηση |
| Παραγόμενο αρχείο | Artifact | ONNX, UBJ, manifest, mapping ή prediction output |
| Αποτύπωμα | Hash digest | Ταυτότητα bytes UBJ με SHA-256 |
| Δημοσίευση | Publication / activation | Αντικατάσταση ενεργών artifacts ή πινάκων |
| Επαναφόρτωση | Reload | Φόρτωση αποθηκευμένου μοντέλου στη μνήμη υπηρεσίας |
| Προσωρινή αποθήκευση | Staging | Ελεγχόμενο ενδιάμεσο πριν από δημοσίευση |
| Αντίθλιψη ροής | Backpressure | Αναμονή παραγωγού όταν ο καταναλωτής δεν προλαβαίνει |
| Αμοιβαίος αποκλεισμός | Mutual exclusion | Προστασία κοινών μεταβλητών ή buffers |
| Δεξαμενή συνδέσεων | Connection pool | Επαναχρησιμοποίηση συνδέσεων PostgreSQL |
| Στιγμιότυπο ανάγνωσης | Read snapshot | Συνεπής όψη δεδομένων μιας συναλλαγής |
| Γεωκωδικοποίηση | Geocoding | Αντιστοίχιση επιλογής διεύθυνσης σε θέση |
| Ορθογώνιο ορίων | Bounding box | Min/max latitude και longitude |
| Σελιδοποίηση | Pagination | Διαίρεση αποτελεσμάτων σε σελίδες |
| Αυθεντικοποίηση | Authentication | Έλεγχος ταυτότητας χρήστη |
| Εξουσιοδότηση | Authorization | Έλεγχος δικαιώματος συγκεκριμένης ενέργειας |
| Ιδιοδυναμία επανάληψης | Idempotency | Ασφαλής επανάληψη ενέργειας χωρίς νέο αποτέλεσμα |

# Πίνακας συντμήσεων και ακρωνυμίων
@table Ανάπτυξη ακρωνυμίων που χρησιμοποιούνται στο κείμενο
| Ακρωνύμιο | Ανάπτυξη | Σημασία |
| --- | --- | --- |
| API | Application Programming Interface | Συμβόλαιο επικοινωνίας εφαρμογών |
| AVM | Automated Valuation Model | Αυτοματοποιημένο μοντέλο εκτίμησης |
| CSV | Comma-Separated Values | Πινακοειδές αρχείο εκπαίδευσης και mappings |
| DDL | Data Definition Language | Ορισμοί σχήματος βάσης |
| EOF | End Of File | Τέλος ροής bytes |
| GIS | Geographic Information System | Γεωγραφικό πληροφοριακό σύστημα |
| HTTP | Hypertext Transfer Protocol | Πρωτόκολλο αιτημάτων υπηρεσιών |
| JSON | JavaScript Object Notation | Μορφή ανταλλαγής δομημένων δεδομένων |
| JWT | JSON Web Token | Υπογεγραμμένο token συνεδρίας |
| MAE | Mean Absolute Error | Μέσο απόλυτο σφάλμα |
| ML | Machine Learning | Μηχανική μάθηση |
| ONNX | Open Neural Network Exchange | Φορητή αναπαράσταση μοντέλου |
| OOF | Out Of Fold | Πρόβλεψη από fit που δεν περιλαμβάνει τη συγκεκριμένη γραμμή |
| OSM | OpenStreetMap | Γεωγραφικός πάροχος και αναγνωριστικά αντικειμένων |
| R² | Coefficient of determination | Συντελεστής προσδιορισμού |
| RMSE | Root Mean Squared Error | Ρίζα μέσου τετραγωνικού σφάλματος |
| SHA-256 | Secure Hash Algorithm, 256 bits | Συνάρτηση αποτυπώματος αρχείου |
| SHAP | SHapley Additive exPlanations | Προσθετικές εξηγήσεις χαρακτηριστικών |
| SQL | Structured Query Language | Γλώσσα σχεσιακών ερωτημάτων |
| SRID | Spatial Reference System Identifier | Αναγνωριστικό συστήματος αναφοράς |
| SVG | Scalable Vector Graphics | Διανυσματική γραφική αναπαράσταση |
| TLS | Transport Layer Security | Ασφάλεια μεταφοράς δικτύου |
| UBJ | Universal Binary JSON | Native δυαδική μορφή μοντέλου XGBoost |
| UID | Unique Identifier | Αναγνωριστικό ακινήτου ή χρήστη |
| ULID | Universally Unique Lexicographically Sortable Identifier | Αναγνωριστικό νέων εγγραφών |

# Βιβλιογραφία
Οι παραπομπές ακολουθούν σύστημα συγγραφέα και έτους. Οι τεχνικές περιγραφές της εφαρμογής προκύπτουν πρωτίστως από το εξεταζόμενο bundle, ενώ οι παρακάτω πηγές θεμελιώνουν έννοιες, συμβάσεις και επιλογές. Οι διαδικτυακές πηγές ελέγχθηκαν κατά τη σύνταξη στις 19/10/2026. Τα παραδείγματα ΔΙΩΝΗ χρησιμοποιήθηκαν για μελέτη οργάνωσης ακαδημαϊκού κειμένου και όχι για μεταφορά αποτελεσμάτων άλλου πεδίου.

Antipov, E. A. και Pokryshevskaya, E. B. (2012). Mass appraisal of residential apartments: An application of Random forest for valuation and a CART-based approach for model diagnostics. Expert Systems with Applications. [Εγγραφή του πρωτότυπου άρθρου](https://www.sciencedirect.com/science/article/pii/S0957417411011894).

Chen, T. και Guestrin, C. (2016). XGBoost: A Scalable Tree Boosting System. Proceedings of KDD 2016. DOI: 10.1145/2939672.2939785. [Πρωτότυπη εργασία](https://arxiv.org/abs/1603.02754).

Go Authors (χ.χ.-α). io package: Reader και ReadCloser. Επίσημη τεκμηρίωση. [Τεκμηρίωση io](https://pkg.go.dev/io).

Go Authors (χ.χ.-β). os/exec package: Cmd, StdoutPipe, Start και Wait. Επίσημη τεκμηρίωση. [Τεκμηρίωση os/exec](https://pkg.go.dev/os/exec).

Go Authors (χ.χ.-γ). bufio package: Scanner και Buffer. Επίσημη τεκμηρίωση. [Τεκμηρίωση bufio](https://pkg.go.dev/bufio).

Go Authors (χ.χ.-δ). sync package: Mutex και RWMutex. Επίσημη τεκμηρίωση. [Τεκμηρίωση sync](https://pkg.go.dev/sync#RWMutex).

Lundberg, S. M. και Lee, S.-I. (2017). A Unified Approach to Interpreting Model Predictions. Advances in Neural Information Processing Systems. [Πρωτότυπη εργασία SHAP](https://arxiv.org/abs/1705.07874).

Lundberg, S. M. και συν. (2020). From local explanations to global understanding with explainable AI for trees. Nature Machine Intelligence. DOI: 10.1038/s42256-019-0138-9. [Δημοσιευμένη εργασία TreeSHAP](https://doi.org/10.1038/s42256-019-0138-9).

OpenStreetMap Foundation (χ.χ.). Nominatim Usage Policy. [Πολιτική της δημόσιας υπηρεσίας](https://operations.osmfoundation.org/policies/nominatim/).

PostGIS Project (χ.χ.). ST_DWithin. Επίσημη τεκμηρίωση λειτουργίας γεωμετρίας και γεωγραφίας. [Τεκμηρίωση χωρικής απόστασης](https://postgis.net/docs/ST_DWithin.html).

PostgreSQL Global Development Group (χ.χ.). PostgreSQL 17 Documentation: Transaction Isolation. [Επίπεδα απομόνωσης συναλλαγών](https://www.postgresql.org/docs/17/transaction-iso.html).

Roberts, D. R. και συν. (2017). Cross-validation strategies for data with temporal, spatial, hierarchical, or phylogenetic structure. Ecography. DOI: 10.1111/ecog.02881. [Πλήρες κείμενο από ερευνητικό φορέα](https://www.wsl.ch/lud/biodiversity_events/papers/Roberts_et_al-2017-Ecography.pdf).

Rosen, S. (1974). Hedonic Prices and Implicit Markets: Product Differentiation in Pure Competition. Journal of Political Economy, 82(1), 34-55. DOI: 10.1086/260169. [Πρωτότυπο άρθρο](https://www.journals.uchicago.edu/doi/10.1086/260169).

scikit-learn Developers (χ.χ.). r2_score. Επίσημη τεκμηρίωση μετρικής παλινδρόμησης. [Ορισμός και ιδιότητες R²](https://scikit-learn.org/stable/modules/generated/sklearn.metrics.r2_score.html).

XGBoost Developers (χ.χ.). Introduction to Model IO. Επίσημη τεκμηρίωση αποθήκευσης JSON/UBJSON και διάκρισης μοντέλων από memory snapshots. [Αποθήκευση μοντέλων](https://xgboost.readthedocs.io/en/stable/tutorials/saving_model.html).

Πανεπιστήμιο Πειραιώς, Τμήμα Πληροφορικής (χ.χ.). Κανονισμός εκπόνησης πτυχιακής εργασίας και ενσωματωμένο πρότυπο συγγραφής. Παρεχόμενο PDF 46 σελίδων· οδηγίες συγγραφής στις σελίδες 12-18 και πρότυπο από τη σελίδα 20. [Ιστοσελίδα κανονισμού Τμήματος](https://cs.unipi.gr/en/kanonismos-ekponhshs-ptyxiakhs-ergasias/). Η παρεχόμενη τοπική έκδοση αποτελεί το υλικό μορφοποίησης της παρούσας εργασίας.

Πάκια, Λ. (2025). Μηχανική μάθηση στην ανίχνευση κυβερνοεπιθέσεων: βιβλιογραφική ανασκόπηση των μεθόδων και των τεχνολογιών. Πτυχιακή εργασία, Τμήμα Πληροφορικής, Πανεπιστήμιο Πειραιώς. [Αποθετήριο ΔΙΩΝΗ](https://dione.lib.unipi.gr/xmlui/handle/unipi/18199).

Φρονίστας, Κ. (2025). Ανάπτυξη διαδραστικού εκπαιδευτικού λογισμικού για την εισαγωγή στον προγραμματισμό με Python. Πτυχιακή εργασία, Τμήμα Πληροφορικής, Πανεπιστήμιο Πειραιώς. [Αποθετήριο ΔΙΩΝΗ](https://dione.lib.unipi.gr/xmlui/handle/unipi/17927).

Πηγαίο υλικό έργου (2026). Property Co-Pilot: Ptixiaki, Ptixiaki-ml-microservice, ml_shared, Ptixiaki-frontend/Ptixiaki-Frontend και schema_dump.sql. Τοπικό bundle που παραδόθηκε για μελέτη· έλεγχος 19/10/2026. Δεν υπάρχει στην παρούσα έκδοση δημόσιο archival DOI ή πλήρες ανωνυμοποιημένο dataset. Οι ακριβείς σχετικές διαδρομές και γραμμές των κρίσιμων αποσπασμάτων αναφέρονται στο σώμα και στο παράρτημα.

# Παράρτημα Α. Συμβόλαια HTTP και διαδρομές
## Α.1 Routes που χρησιμοποιεί η διεπαφή
@table Διαδρομές και αναγνωστικά ή μεταβλητικά συμβόλαια
| Route | Μέθοδος | Είσοδος / πρόσβαση | Βασική έξοδος |
| --- | --- | --- | --- |
| /register/SignIn | POST | email, password | Cookie και αποτέλεσμα σύνδεσης |
| /register/SignUp | POST | Στοιχεία εγγραφής | Cookie και αποτέλεσμα εγγραφής |
| /User/BrowseProperties | GET | JWT, filters, index | index, page_size, has_next, properties |
| /User/BrowseProperty | GET | JWT, uid | Κοινό detail αγοράς |
| /User/Portfolio | GET | JWT | property_count, total_estimated_value, properties |
| /User/ViewProperty | GET | JWT, uid, ownership | Owner detail |
| /User/AddProperty | POST | JWT, transaction, JSON | 201, message, uid, Location |
| /User/DeleteProperty | DELETE | JWT, property_uid, ownership | Μήνυμα αφαίρεσης σχέσης |
| /analysis/autocomplete | GET | JWT, q, optional kind=address | Προτάσεις περιοχής ή διεύθυνσης |
| /analysis/CreateAnalysisForArea | POST | JWT, αριθμητικό bbox | Points, Clusters, SalesReport, RentReport, warnings |

Ο πίνακας περιγράφει τις μεθόδους χρήσης από τη διεπαφή. Δεν δηλώνει ότι όλοι οι παλαιότεροι auth handlers επιβάλλουν τις ίδιες method/body validation δικλίδες με τους νεότερους. Η owner detail διατηρεί επίσης legacy POST μορφή με PropertyUid. Τα /train, /test και /predict/sell υπάρχουν στην κύρια υπηρεσία αλλά δεν αποτελούν τις κανονικές προστατευμένες ενέργειες διεπαφής. Η δημόσια έκθεσή τους στην τρέχουσα μορφή δεν συνιστάται.

## Α.2 Εσωτερικό συμβόλαιο Python
@table Αιτήματα προς τη μικροϋπηρεσία
| Endpoint | Είσοδος | Μοντέλο | Αποτέλεσμα |
| --- | --- | --- | --- |
| /ReloadExplainModels | transactions, expected_model_sha256 | Φόρτωση από κοινό filesystem | Summaries ενεργών native μοντέλων |
| /ExplainProperty | transaction, features | Ήδη φορτωμένο UBJ | Baseline, contributions, prediction, hash |
| /TrainAndExplain | data, target, categories, dropped columns, params | Νέο προσωρινό XGBoost | R², importance, interactions, 10 examples |

Το batch SHAP της καθολικής εκπαίδευσης δεν περνά από /ExplainProperty. Παράγεται μέσα στο Tree.py και μεταφέρεται μέσω stdout προς τη Go. Η διάκριση αποφεύγει χιλιάδες HTTP κλήσεις κατά την εκπαίδευση. Το /ExplainProperty προορίζεται για νέα μεμονωμένη αριθμητική είσοδο στην ήδη φορτωμένη γενιά.

## Α.3 Παράδειγμα μορφής επιτυχίας προσθήκης
Το ακόλουθο JSON δείχνει τη μορφή και όχι πραγματική νέα καταχώριση. Το UID πρέπει να ληφθεί από την πραγματική απόκριση. Δεν είναι προέκταση του request και δεν παράγεται από τον browser.

```json
{
  "message": "Property added successfully",
  "uid": "property_<generated-ulid>"
}
```

# Παράρτημα Β. Ολοκληρωμένος κώδικας κρίσιμης ροής
## Β.1 Εκτέλεση trainer και δημοσίευση SHAP
Παρατίθεται ολόκληρη η RunTrainingAndPersist, ώστε τα επιμέρους αποσπάσματα του κεφαλαίου 7 να διαβάζονται και ως ενιαία ροή. Τα αμιγώς σχολιαστικά lines παραλείπονται για οικονομία χώρου, αλλά διατηρούνται οι αριθμοί γραμμών του αρχείου. Το απόσπασμα απαιτεί το υπόλοιπο package και δεν είναι αυτόνομο πρόγραμμα. Το context ελέγχει τη διάρκεια, ο parser την εγκυρότητα και η συναλλαγή τη δημοσίευση των συγκεκριμένων πινάκων.

@source Ptixiaki/Worker/explainability/training.go|63|182

## Β.2 Πλήρης μαθηματικός ορισμός του score στον κοινό helper
Οι helpers χρησιμοποιούνται ως κοινός ορισμός της τυποποίησης και της βαθμολογίας. Η απλότητα αυτών των συναρτήσεων διευκολύνει τη δοκιμή κατεύθυνσης των προσήμων. Δεν αρκεί όμως για ισότητα δύο πλήρων pipelines όταν διαφέρουν τα σύνολα γειτόνων ή τα αποθηκευμένα στατιστικά τους.

@source Ptixiaki/Worker/Pipelines/CalculateEloForProps.go|198|207

## Β.3 Αναπαραγώγιμη επιλογή δείγματος αλληλεπιδράσεων
Η παρακάτω συνάρτηση δείχνει όλο τον μηχανισμό επιλογής. Η ταξινόμηση πριν από την ψευδοτυχαία δειγματοληψία προστατεύει από αλλαγή σειράς δεδομένων. Το seed και το πλήθος αποθηκεύονται ως μεταδεδομένα, ώστε η διαδικασία να είναι ελέγξιμη.

@source Ptixiaki/Worker/Models/TreeCode/Tree.py|95|108

# Παράρτημα Γ. Χάρτης ανάγνωσης και αναπαραγωγής
## Γ.1 Από το σύμπτωμα στο αρχείο
@table Αφετηρίες ανάγνωσης πηγαίου κώδικα
| Θέμα | Αφετηρία πηγαίου κώδικα |
| --- | --- |
| Τι ξεκινά πρώτο | Ptixiaki/main.go: main, WarmUpApplication, WarmUpTrees |
| Πώς εκπαιδεύονται δύο αγορές | main.go: trainResidenceTrees και Worker/Pipelines/TrainingPipeline.go |
| Ποιες στήλες μπαίνουν στο μοντέλο | Worker/training, Worker/testing/CalculatePrice.go και explainability/contract.go |
| Πώς δημιουργείται manifest | Tree.py: stage_native_explainability_artifacts και ml_shared/explain_artifacts.py |
| Ποιος διαβάζει το stdout | Worker/explainability/training.go και records.go |
| Πώς τυπώνεται πρόοδος | Worker/explainability/progress.go |
| Πού βρίσκει UBJ η Python | Ptixiaki-ml-microservice/main.py: discover_model_roots |
| Πώς προστίθεται ακίνητο | Backend/Api/UserActions/AddPropertyHandler.go, UserActions.go, DataBaseActions.go |
| Πώς διαβάζεται πλήρες detail | Backend/Api/UserActions/PropertyDetails.go |
| Πώς γίνεται pagination | Backend/Api/UserActions/BrowseProperties.go |
| Πώς επιλέγεται περιοχή | MarketAnalysis/AreaSearch.go και frontend AreaSearch.tsx |
| Πώς δημιουργούνται clusters | MarketAnalysis/Kmeans.go και AnalysisActions.go |
| Πώς αλλάζει σελίδα το React | src/components/Workspace.tsx |

## Γ.2 Εντολές επαλήθευσης
Οι εντολές δεν περιλαμβάνουν ενεργοποίηση προαιρετικών live database tests και δεν καλούν /train. Το Go build cache παραμένει εκτός του bundle, στην προεπιλεγμένη διαδρομή των Windows `%LOCALAPPDATA%\go-build`. Αν η διαδρομή δεν είναι εγγράψιμη, επιλέγεται προσωρινός κατάλογος του λειτουργικού συστήματος εκτός του bundle. Δεν επαναδημιουργούνται οι κατάλογοι `.gocache` και `.cache` μέσα στο έργο. Η Python εντολή πρέπει να εκτελείται με interpreter που διαθέτει και Worker εξαρτήσεις, όχι με ελλιπές service-only περιβάλλον.

```powershell
Set-Location S:\ptixiaki-bundle\Ptixiaki
$env:GOCACHE = Join-Path $env:LOCALAPPDATA 'go-build'
go test ./...
```

```powershell
Set-Location S:\ptixiaki-bundle\Ptixiaki-frontend\Ptixiaki-Frontend
npm run lint
npm run build
npm test
```

```powershell
Set-Location S:\ptixiaki-bundle\Ptixiaki-ml-microservice
python -m unittest -v test_explainability.py
```

## Γ.3 Τι απαιτείται για πλήρη επιστημονική αναπαραγωγή
Η αναπαραγωγή μοντέλων απαιτεί περισσότερα από την επιτυχή εκτέλεση των εντολών δοκιμών. Χρειάζεται συγκεκριμένο ανωνυμοποιημένο snapshot δεδομένων, τεκμηρίωση δικαιωμάτων χρήσης, καθαρό πρωτόκολλο split, εκδόσεις interpreters και native runtime, πλήρη manifests και ελεγχόμενη διαμόρφωση βάσης. Κανένα μυστικό ή session αρχείο δεν πρέπει να περιλαμβάνεται στο συνοδευτικό υλικό. Το source bundle πρέπει να καθαριστεί από τέτοια στοιχεία πριν κατατεθεί ή δημοσιευθεί.

