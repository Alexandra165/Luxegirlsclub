LuxGirlsClub

LuxeGirlsClub is a fully deployed multilingual web platform for managing user profiles. It is built from scratch and maintained by a single developer. The project includes full backend, frontend, cloud integration, admin tools, and SEO.

Live: https://luxgirlsclub.com

Stack

– Go (Golang) backend  
– HTML, CSS, Bootstrap, JavaScript frontend  
– MySQL database  
– Cloud storage via Backblaze B2 (S3-compatible)  
– VPS hosting with Apache + NGINX (Ubuntu)  
– Cloudflare CDN and DDoS protection  
– Responsive design for desktop and mobile  
– Custom admin panel with moderation tools

Key Features

– User registration, login, and profile management  
– Upload and display of photos and videos (cloud-based)  
– Filtering by country, city, and district  
– Online status tracking and verified profile badges  
– Multilingual UI: English, Russian, Georgian  
– Manual admin moderation and payment tracking  
– SEO-optimized city-based landing pages  
– IP geolocation-based content filtering

What I Did Personally

– Full-stack development (frontend and backend)  
– Designed and implemented the database schema  
– Built REST endpoints and profile logic in Go  
– Integrated media upload using AWS SDK and Backblaze B2  
– Created mobile-friendly interface with dynamic JavaScript logic  
– Set up and secured VPS, domains, and Cloudflare  
– Built admin panel for profile control and moderation  
– Wrote all SEO metadata, city landing pages, and handled Google indexing  
– Maintained the system, fixed bugs, and optimized for performance

How to Run Locally

1. Install Go and MySQL  
2. Clone the repository  
   `git clone https://github.com/Alexandra165/Luxgirlsclub.git`  
3. Create a `.env` file with your local config (see `.env.example`)  
4. Run backend:  
   `go run backend/main.go`  
5. Open `index.html` in a browser

Author

Alexandra Tovmasyan  
https://github.com/Alexandra165
