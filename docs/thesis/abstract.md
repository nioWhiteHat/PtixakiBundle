Η εργασία παρουσιάζει το Property Co-Pilot, ένα διαδικτυακό σύστημα υποστήριξης αποφάσεων για κατοικίες πώλησης και μίσθωσης. Κίνητρο είναι η μείωση της πληροφοριακής ασυμμετρίας: η δυνατότητα σύγκρισης αγγελιών, εντοπισμού πιθανής υπερτιμολόγησης και αποφυγής δυσμενών συμφωνιών με περισσότερα τεκμηριωμένα στοιχεία. Η εφαρμογή δεν πιστοποιεί αξίες ή προθέσεις πωλητών και δεν αποδεικνύει συνολική διόρθωση της αγοράς.

Το σύστημα συνδυάζει React/TypeScript, Go, PostgreSQL/PostGIS και Python/FastAPI. Δύο φάσεις XGBoost συνδέουν χαρακτηριστικά κατοικίας με χωρικά κατάλοιπα. ONNX και UBJ εξυπηρετούν αντίστοιχα πρόβλεψη και native SHAP, ενώ manifests και hashes συντονίζουν τη φόρτωση. Η μελέτη αναλύει την αρχικοποίηση, τα συμβόλαια, την αποθήκευση, την ευρετική κατάταξη και τα προσωρινά μοντέλα περιοχής. Οι διαθέσιμες μετρικές δείχνουν ασθενή επικύρωση μισθώσεων και η target-derived χωρική πληροφορία απαιτεί καθαρότερο πειραματικό διαχωρισμό. Η συμβολή είναι η λειτουργική ολοκλήρωση εξηγήσιμης ανάλυσης με ρητή καταγραφή των ορίων της.

Επιστημονική περιοχή: Εφαρμοσμένη μηχανική μάθηση και μηχανική λογισμικού.

Λέξεις-κλειδιά: ακίνητα, XGBoost, SHAP, γεωχωρική ανάλυση, υποστήριξη αποφάσεων.

## Abstract
This thesis presents Property Co-Pilot, a web-based decision-support system for residential sale and rental listings. Its motivation is to reduce information asymmetry by enabling comparable listing views, identifying potentially excessive asking prices and supporting better-informed negotiation. It does not certify market value, infer seller misconduct or demonstrate a market-wide price correction.

The system combines React/TypeScript, Go, PostgreSQL/PostGIS and Python/FastAPI. A two-stage XGBoost pipeline enriches property attributes with spatial residuals. ONNX serves inference, native UBJ models support SHAP explanations, and manifests with hashes coordinate loading. The study examines initialization, data contracts, persistence, heuristic ranking and temporary area models. Stored metrics reveal weak rental validation, while target-derived spatial features require a cleaner evaluation protocol. The contribution is an integrated explainable analysis workflow with explicit numerical, operational and scientific limitations.

Scientific field: Applied machine learning and software engineering.

Keywords: real estate, XGBoost, SHAP, geospatial analysis, decision support.
