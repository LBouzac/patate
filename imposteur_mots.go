package main

// Thèmes de l'Imposteur : 12 mots chacun. L'imposteur démasqué doit retrouver le mot secret parmi les 12.
var impBank = []struct {
	Cat   string
	Words [12]string
}{
	{"Animaux", [12]string{"Chat", "Chien", "Lion", "Éléphant", "Girafe", "Pingouin", "Dauphin", "Serpent", "Lapin", "Cheval", "Singe", "Crocodile"}},
	{"Plats", [12]string{"Pizza", "Croissant", "Sushi", "Hamburger", "Crêpe", "Raclette", "Couscous", "Lasagnes", "Fondue", "Baguette", "Tacos", "Quiche"}},
	{"Fruits", [12]string{"Pomme", "Banane", "Fraise", "Ananas", "Pastèque", "Cerise", "Mangue", "Kiwi", "Citron", "Raisin", "Orange", "Poire"}},
	{"Métiers", [12]string{"Pompier", "Médecin", "Boulanger", "Policier", "Professeur", "Pilote", "Cuisinier", "Dentiste", "Plombier", "Astronaute", "Coiffeur", "Facteur"}},
	{"Sports", [12]string{"Football", "Tennis", "Natation", "Ski", "Boxe", "Rugby", "Basket", "Judo", "Cyclisme", "Surf", "Escalade", "Golf"}},
	{"Lieux", [12]string{"Plage", "Hôpital", "Cinéma", "Aéroport", "Bibliothèque", "Piscine", "Supermarché", "École", "Musée", "Gare", "Prison", "Zoo"}},
	{"Objets de la maison", [12]string{"Frigo", "Canapé", "Lampe", "Miroir", "Aspirateur", "Télévision", "Oreiller", "Horloge", "Parapluie", "Ciseaux", "Brosse à dents", "Tapis"}},
	{"Transports", [12]string{"Voiture", "Vélo", "Avion", "Bateau", "Train", "Métro", "Trottinette", "Hélicoptère", "Moto", "Fusée", "Tramway", "Montgolfière"}},
	{"Pays", [12]string{"France", "Japon", "Brésil", "Italie", "Canada", "Égypte", "Australie", "Mexique", "Inde", "Espagne", "Norvège", "Maroc"}},
	{"Instruments de musique", [12]string{"Piano", "Guitare", "Violon", "Batterie", "Trompette", "Flûte", "Harpe", "Saxophone", "Accordéon", "Triangle", "Violoncelle", "Harmonica"}},
	{"Vêtements", [12]string{"Chaussettes", "Écharpe", "Casquette", "Pyjama", "Cravate", "Manteau", "Short", "Gants", "Bottes", "Maillot de bain", "Pull", "Robe"}},
	{"Événements", [12]string{"Anniversaire", "Mariage", "Noël", "Halloween", "Carnaval", "Pique-nique", "Concert", "Vacances", "Déménagement", "Examen", "Rentrée", "Réveillon"}},
	{"Nature", [12]string{"Volcan", "Forêt", "Désert", "Cascade", "Montagne", "Île", "Glacier", "Grotte", "Rivière", "Océan", "Prairie", "Marais"}},
	{"Jeux", [12]string{"Échecs", "Cache-cache", "Puzzle", "Dominos", "Loto", "Cartes", "Billard", "Bowling", "Karaoké", "Fléchettes", "Marelle", "Jeu de l'oie"}},
	{"Corps humain", [12]string{"Nez", "Genou", "Coude", "Oreille", "Épaule", "Cheville", "Nombril", "Sourcil", "Poignet", "Menton", "Langue", "Orteil"}},
	{"Ustensiles de cuisine", [12]string{"Fourchette", "Casserole", "Fouet", "Louche", "Poêle", "Couteau", "Passoire", "Râpe", "Rouleau à pâtisserie", "Tire-bouchon", "Spatule", "Saladier"}},
	{"Desserts", [12]string{"Tiramisu", "Éclair", "Macaron", "Mousse au chocolat", "Tarte aux pommes", "Crème brûlée", "Glace", "Gaufre", "Beignet", "Profiterole", "Flan", "Cookie"}},
	{"Créatures fantastiques", [12]string{"Dragon", "Sirène", "Vampire", "Fantôme", "Licorne", "Zombie", "Sorcière", "Loup-garou", "Elfe", "Ogre", "Fée", "Extraterrestre"}},
	{"Météo", [12]string{"Pluie", "Neige", "Orage", "Arc-en-ciel", "Brouillard", "Tempête", "Grêle", "Canicule", "Tornade", "Verglas", "Nuage", "Éclair"}},
	{"À l'école", [12]string{"Cartable", "Tableau", "Récréation", "Cantine", "Dictée", "Gomme", "Règle", "Trousse", "Compas", "Cahier", "Bulletin", "Surveillant"}},
}
