---
name: vehicle-selection
description: Guidance for selecting one concrete rental vehicle from destination, trip type, passenger count, and budget tier.
---

# Vehicle Selection Guidance

Recommend one specific make and model. Do not answer with only a category such as "SUV", "MPV", or "sedan". Do not reuse one default vehicle for every trip. Volkswagen, BMW, and a generic family MPV are not the fallback.

Read the budget tier from the first word of the budget (`economy`, `moderate`, `comfortable`, or `luxury`). Then use the first matching row below. The `model` must be the name in that cell. The `type` must be one of `compact`, `estate`, `suv`, `mpv`, or `convertible`.

## Destination, 1–4 travelers

| Destination mentions | economy | moderate | comfortable | luxury |
| --- | --- | --- | --- | --- |
| Italy, Rome, Florence, Amalfi, Riviera, Sicily, Tuscany | Fiat Panda | Jeep Renegade | Alfa Romeo Tonale | Alfa Romeo Giulia |
| France, Paris, Lyon, Provence, Nice | Renault Clio | Peugeot 3008 | Citroen C5 Aircross | DS 7 |
| Spain, Barcelona, Madrid, Andalusia, Seville | Seat Ibiza | Seat Leon | Cupra Formentor | Audi Q5 |
| Germany, Austria, Bavaria, Alps, Dolomites | Skoda Fabia | Skoda Octavia | BMW X1 | BMW 5 Series |
| Switzerland | Dacia Sandero | Skoda Kamiq | Volvo XC60 | Volvo XC90 |
| United Kingdom, London, Scotland, Ireland | Vauxhall Corsa | Mini Clubman | Range Rover Evoque | Jaguar XF |
| Norway, Iceland, fjord | Dacia Duster | Subaru Forester | Volvo XC60 | Volvo XC90 |
| Chile, Andes, Patagonia, Atacama, Santiago | Chevrolet Onix | Toyota RAV4 | Toyota Fortuner | Toyota Land Cruiser |
| Argentina, Buenos Aires, Mendoza, Bariloche | Fiat Cronos | Volkswagen Nivus | Toyota SW4 | Toyota Land Cruiser |
| Portugal, Lisbon, Porto | Renault Clio | Peugeot 2008 | Peugeot 3008 | Mercedes GLC |

## Five or more travelers

Ignore the table above and use this row for the same destination group:

| Destination group | economy | moderate | comfortable | luxury |
| --- | --- | --- | --- | --- |
| Italy or France or Spain or Portugal | Fiat Tipo Wagon | Peugeot 5008 | Citroen SpaceTourer | Mercedes V-Class |
| Germany, Austria, Switzerland, UK, Norway | Dacia Jogger | Volkswagen Touran | Volkswagen Multivan | Mercedes V-Class |
| Chile or Argentina | Chevrolet Spin | Toyota Avanza | Chevrolet Trailblazer | Toyota Land Cruiser |

`type` for these is `mpv`, except Toyota Land Cruiser and Chevrolet Trailblazer, which are `suv`.

## No destination row matched

Use the trip type and budget:

| Trip type | economy | moderate | comfortable | luxury |
| --- | --- | --- | --- | --- |
| family, 1–2 travelers | Dacia Logan | Skoda Octavia Combi | Volkswagen Passat Variant | Volvo V90 |
| family, 3–4 travelers | Dacia Duster | Toyota Corolla Cross | Toyota RAV4 | Volvo XC60 |
| adventure | Suzuki Vitara | Subaru Forester | Toyota Land Cruiser Prado | Toyota Land Cruiser |
| business, 1 traveler | Toyota Yaris | Audi A3 | Mercedes C-Class | Audi A6 |
| business, 2 or more | Skoda Scala | Skoda Superb | Audi A6 | BMW 5 Series |
| romantic, 1–2 travelers | Fiat 500C | Mini Cooper Cabrio | BMW 4 Series Convertible | Mercedes CLE Cabriolet |
| roadtrip | Mazda 3 | Mazda 6 | Volkswagen Arteon | Audi A6 Avant |

Family and road-trip cars in this table are `estate` when the name says Wagon, Combi, Variant, or Avant; otherwise `suv` for Duster, RAV4, XC60, and Forester. Romantic cars for 1–2 travelers are `convertible`. Business cars are `compact`. Adventure cars are `suv`.

## Preferences override the row

Apply only the first matching preference:

- Electric or EV, and the destination is a city or a major motorway corridor: Fiat 500e (economy), Volvo EX30 (moderate), BMW iX1 (comfortable), Mercedes EQE (luxury). `type` is `compact`. Do not pick an EV for rural, mountain, or Patagonia routes.
- Convertible, cabrio, or open-top, and travelers are 1 or 2: use the romantic row. Do not recommend a convertible for 3 or more travelers.
- Ski, snow, or winter: Subaru Forester (economy or moderate) or Volvo XC60 (comfortable or luxury). `type` is `suv`.
- Bike, surf, or a lot of luggage, with 1–4 travelers: Skoda Octavia Combi (economy or moderate) or Volkswagen Passat Variant (comfortable or luxury). `type` is `estate`.

## Rules

- Automatic gearbox for cities and mountain roads.
- Full 4WD only for tracks, winter passes, fjords, or the Andes. Say that in the reasoning.
- Economy must not be a luxury or sports brand.
- In the reasoning, name the destination cue, trip type, traveler count, and budget tier that selected this model.
