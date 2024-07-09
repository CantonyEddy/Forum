# Pickaxes

Ce projet a pour but de créer un forum sur Docker avec comme thème le jeu Minecraft.

## Fonctionnalités

- Créer un compte utilisateur.
- Se connecter via un compte Google ou GitHub.
- Trois niveaux d'utilisateur :
  - **Utilisateur "Anonyme"** : Peut voir les postes mais ne peut pas interagir.
  - **Utilisateur "Connecté"** : Peut voir, écrire et interagir avec les postes.
  - **Utilisateur "Admin"** : Peut voir, créer et supprimer des postes.
- Voir les postes existants.
- Créer des postes et supprimer les nôtres.
- Filtrer les postes par catégories.
- Créer des commentaires sous les postes.
- Liker ou disliker des postes.
- Accéder à différentes pages selon les droits de l'utilisateur :
  - `login`
  - `register`
  - `forumMainPage`
  - `createPost`
  - `post`
  - `profile`
  - `adminPannel`
- Les utilisateurs Admin peuvent supprimer n'importe quel poste.
- Le projet peut être lancé avec Docker.

## Installation

Pour installer et lancer le projet, suivez les étapes suivantes :

1. Clonez le dépôt git avec le lien suivant :
   ```sh
   git clone https://github.com/RYUJINC/Forum.git
   ```
2. Modifiez le fichier Docker pour y mettre le chemin absolu de l'emplacement du projet sur votre ordinateur.
3. Dans votre terminal d'IDE, exécutez les commandes suivantes :
   ```sh
   docker build -t forum .
   docker run -p 8080:8080 forum
   ```

## Usage

Les utilisateurs peuvent utiliser le projet pour créer des postes sur Minecraft et partager leurs découvertes ou autres informations.

## Contribution

Nous ne souhaitons pas de contributions externes à ce projet.

## Licence

Ce projet n'est pas destiné à être modifié par des inconnus.

## Auteurs

- Eddy Cantony
- Alenzo Amico
- Alexandre Echazarreta
- Aymeric Moncla

## Contact
Pour toute question ou suggestion, veuillez ouvrir une issue sur le dépôt GitHub ou me contacter par discord via le server [KALIX](https://discord.gg/Dmh6wHaKvD)
