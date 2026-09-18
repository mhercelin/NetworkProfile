# Code signing policy

*Politique de signature de code — version française plus bas.*

Free code signing provided by [SignPath.io](https://signpath.io), certificate by
[SignPath Foundation](https://signpath.org).

## Team roles

- **Committers and reviewers:** [Maxime HERCELIN](https://github.com/mhercelin)
- **Approvers:** [Maxime HERCELIN](https://github.com/mhercelin)

This is a single-maintainer project. Every change is committed, reviewed and
approved by the same person, and the source of every signed build is the public
history of this repository.

## Privacy

This program will not transfer any information to other networked systems
unless specifically requested by the user or the person installing or operating
it.

More precisely, NetworkProfile:

- stores its profiles in a single YAML file under `%AppData%\NetworkProfile`,
  and writes a local log beside it;
- reads the network adapters of the machine it runs on, and the wireless
  networks Windows already knows, to display them;
- never sends anything anywhere: there is no telemetry, no update check, no
  crash reporting and no network request of its own. The fonts it displays are
  embedded in the executable rather than fetched, precisely so that it works on
  a machine whose connection is broken;
- uses no third-party service.

The only network traffic the program causes is the reconfiguration of the
adapter you asked it to change.

---

# Politique de signature de code

Signature de code fournie gratuitement par [SignPath.io](https://signpath.io),
certificat délivré par la [SignPath Foundation](https://signpath.org).

## Rôles

- **Auteur et relecteur des modifications :** [Maxime HERCELIN](https://github.com/mhercelin)
- **Approbateur des signatures :** [Maxime HERCELIN](https://github.com/mhercelin)

Le projet n'a qu'un mainteneur. Chaque modification est écrite, relue et
approuvée par la même personne, et la source de chaque binaire signé est
l'historique public de ce dépôt.

## Confidentialité

Ce programme ne transmet aucune information à d'autres systèmes en réseau, sauf
demande explicite de l'utilisateur ou de la personne qui l'installe ou l'exploite.

Concrètement, NetworkProfile :

- conserve ses profils dans un unique fichier YAML sous `%AppData%\NetworkProfile`,
  et écrit un journal local à côté ;
- lit les cartes réseau de la machine et les réseaux sans fil déjà connus de
  Windows, pour les afficher ;
- n'émet rien : aucune télémétrie, aucune vérification de mise à jour, aucun
  rapport d'incident, aucune requête réseau propre. Les polices de caractères
  sont embarquées dans l'exécutable plutôt que téléchargées, précisément pour
  que l'outil fonctionne sur une machine dont la connexion est en panne ;
- n'utilise aucun service tiers.

Le seul trafic réseau que le programme provoque est la reconfiguration de la
carte que vous lui avez demandé de modifier.
