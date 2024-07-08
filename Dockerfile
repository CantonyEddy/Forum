FROM golang:1.21

# Définir le répertoire de travail
WORKDIR C:/Users/vieil/Documents/cours git/forum/Forum

# Copier les fichiers go.mod et go.sum
COPY go.mod go.sum ./

# Télécharger les dépendances
RUN go mod download && go mod verify

# Copier le reste des fichiers de l'application
COPY . .

# Construire l'application
RUN go build -o main ./main.go

# Exposer le port 8080
EXPOSE 8080

# Exposer le port 443
EXPOSE 443

# Spécifier la commande à exécuter lorsque le conteneur démarre
CMD ["./main"]
