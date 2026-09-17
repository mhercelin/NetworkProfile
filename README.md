# NetworkProfile

Bascule rapide entre configurations réseau sous Windows 11.

Enregistrez une configuration IP par site, par machine ou par équipement, puis
passez de l'une à l'autre en un clic — sans rouvrir les panneaux réseau de
Windows, et sans retaper une adresse de mémoire.

L'outil est né d'un besoin concret : configurer des équipements industriels qui
vivent chacun sur sa propre plage d'adresses (`192.168.0.x` pour un automate,
`10.10.128.x` sur un site client, `172.16.x.x` sur un chantier), plusieurs fois
par jour, sur une machine dont on n'est pas administrateur.

## Ce qu'il fait

- **Profils multi-cartes.** Un profil porte une liste de cartes cibles, pas une
  seule. Le profil « Défaut » remet ainsi l'Ethernet *et* le Wi-Fi en DHCP d'un
  seul geste.
- **Adresse fixe ou DHCP**, avec adresse, masque, passerelle et plusieurs DNS.
- **Wi-Fi** : bascule vers un réseau déjà enregistré par Windows, ou changement
  d'adresse en restant sur le réseau courant. Les clés ne sont jamais
  manipulées : c'est Windows qui les détient.
- **Changement rapide** : appliquer une adresse ponctuelle sans créer de profil.
- **Cartes virtuelles masquées** par défaut — Hyper-V, WSL et VMware ne
  polluent pas le sélecteur, mais restent affichables.
- **Profil actif signalé** : la ligne dont la configuration correspond à l'état
  réel des cartes est mise en évidence.
- **Reste en zone de notification**, pour ne demander les droits qu'une fois.

## Le modèle de privilèges

C'est la partie non évidente du projet, et elle mérite d'être comprise avant de
toucher au code.

Reconfigurer une carte réseau exige les droits administrateur. Mais sur un poste
d'entreprise, l'utilisateur n'est **pas** administrateur : élever un programme y
signifie basculer sur un **compte différent**. Une application entièrement
élevée y perd donc son propre profil utilisateur — `%AppData%` désigne alors le
compte administrateur, et les profils réseau disparaissent d'un lancement à
l'autre.

L'application est donc coupée en deux :

```
NetworkProfile.exe              votre compte, aucune élévation
  ├─ profils        %AppData%\NetworkProfile\profiles.yaml
  ├─ lecture état   API Windows, directe — aucun privilège requis
  └─ écriture ──────┐
                    │  tube nommé, nom aléatoire, accès restreint à votre compte
NetworkProfile.exe --helper     élevé, lancé à la première modification
  └─ netsh
```

Conséquences pratiques :

- Ouvrir l'application et consulter l'état du réseau ne demande **rien**.
- Les identifiants administrateur sont demandés **au premier changement**, puis
  plus du tout tant que l'application reste ouverte — d'où l'icône en zone de
  notification, qui la garde vivante après fermeture de la fenêtre.
- L'assistant élevé surveille la fenêtre qui l'a lancé et **s'arrête avec
  elle** : aucun processus privilégié ne subsiste.
- Son tube porte un nom aléatoire et une liste de contrôle d'accès protégée ne
  nommant que votre compte, SYSTEM et les administrateurs : aucun autre
  utilisateur du poste ne peut lui parler.

## Format des profils

`%AppData%\NetworkProfile\profiles.yaml`, lisible et modifiable à la main :

```yaml
version: 1
profiles:
  - id: ethernet-automate-s7
    name: Ethernet — Automate S7-1500
    targets:
      - interface: Ethernet
        mode: static
        address: 192.168.0.241
        mask: 255.255.255.0

  - id: wifi-site-client-b
    name: Wi-Fi — Site client B
    targets:
      - interface: Wi-Fi
        ssid: SiteB-Corp        # réseau déjà connu de Windows
        mode: static
        address: 10.10.128.20
        mask: 255.255.255.0
        gateway: 10.10.128.1
        dns: [10.10.128.1, 9.9.9.9]

  - id: defaut-dhcp
    name: Défaut — DHCP
    targets:
      - interface: Ethernet
        mode: dhcp
      - interface: Wi-Fi
        mode: dhcp
```

Une passerelle hors du sous-réseau de l'adresse est refusée : Windows l'accepte
et route ensuite de façon imprévisible, ce qui coûte une heure de diagnostic.

## Construire

Nécessite [Go](https://go.dev/dl/), [Node.js](https://nodejs.org/) et la
[CLI Wails](https://wails.io/docs/gettingstarted/installation).

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails build
```

Le binaire atterrit dans `build/bin/NetworkProfile.exe`.

Pour itérer sur l'interface, `wails dev` doit tourner dans un **terminal
administrateur** : le binaire de développement hérite du manifeste et refuse
sinon de démarrer.

L'icône est dessinée par programme — pour la modifier, éditez les coordonnées
dans `tools/icongen/main.go` puis :

```bash
go run tools/icongen/main.go
```

## Tests

```bash
go test ./...
cd frontend && npm test
```

Les tests Go pilotent une fausse couche réseau : aucun n'altère la configuration
de la machine. Ceux qui touchent de vraies cartes sont derrière une étiquette de
compilation et ne partent jamais tout seuls :

```bash
go test -tags integration ./internal/network -run TestRealAdapters -v
```

## Licence

À définir.
