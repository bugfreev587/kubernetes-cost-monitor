# k8s-cost - Authentication Service Frontend

A modern React application for user authentication and management, built with Clerk authentication service. Features a clean, responsive UI with signup, signin, and dashboard functionality.

## Features

- 🔐 **User Authentication** - Secure signup and signin powered by Clerk
- 🏠 **Homepage** - Landing page with navigation and call-to-action
- 📊 **Dashboard** - Protected user dashboard displaying profile information
- 🧭 **Navigation** - Responsive navbar with Features, Pricing, and user menu
- 🔒 **Protected Routes** - Automatic redirects based on authentication state
- 📱 **Responsive Design** - Mobile-friendly UI that works on all devices

## Tech Stack

- **React 19** - UI library
- **TypeScript** - Type-safe JavaScript
- **Vite** - Build tool and dev server
- **React Router** - Client-side routing
- **Clerk** - Authentication service
- **CSS3** - Styling with modern gradients and animations

## Getting Started

### Prerequisites

- Node.js (v18 or higher)
- npm or yarn

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd auth-service-frontend
```

2. Install dependencies:
```bash
npm install
```

3. Create a `.env` file in the root directory:
```env
VITE_CLERK_PUBLISHABLE_KEY=your_clerk_publishable_key_here
```

4. Start the development server:
```bash
npm run dev
```

The application will be available at `http://localhost:5173`

## Project Structure

```
auth-service-frontend/
├── src/
│   ├── components/
│   │   └── Navbar.tsx          # Navigation bar component
│   ├── pages/
│   │   ├── HomePage.tsx        # Landing page
│   │   ├── SignInPage.tsx      # Sign in page
│   │   ├── SignUpPage.tsx      # Sign up page
│   │   ├── Dashboard.tsx       # Protected dashboard
│   │   ├── FeaturesPage.tsx   # Features page
│   │   └── PricingPage.tsx     # Pricing page
│   ├── App.tsx                 # Main app component with routing
│   ├── App.css                 # Application styles
│   ├── index.css               # Global styles
│   └── main.tsx                # Entry point
├── public/                     # Static assets
├── vercel.json                 # Vercel deployment configuration
└── package.json                # Dependencies and scripts
```

## Routes

- `/` - Homepage (redirects authenticated users to `/dashboard`)
- `/sign-in` - Sign in page
- `/sign-up` - Sign up page
- `/dashboard` - Protected dashboard (requires authentication)
- `/features` - Features page
- `/pricing` - Pricing page

## Environment Variables

Create a `.env` file with the following variable:

- `VITE_CLERK_PUBLISHABLE_KEY` - Your Clerk publishable key

**Note:** The `.env` file is already added to `.gitignore` to keep your keys secure.

## Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build locally
- `npm run lint` - Run ESLint

## Deployment

### Vercel

The project includes a `vercel.json` configuration file for proper SPA routing on Vercel. The configuration ensures that all routes are properly handled by React Router.

To deploy:

1. Push your code to a Git repository
2. Import the project in Vercel
3. Add your `VITE_CLERK_PUBLISHABLE_KEY` environment variable in Vercel settings
4. Deploy

The `vercel.json` file automatically handles client-side routing, preventing 404 errors on direct route access.

## Authentication Flow

1. **Unauthenticated Users:**
   - Can access homepage, Features, and Pricing pages
   - Redirected to sign-in when accessing protected routes
   - Can sign up or sign in via the navigation bar

2. **Authenticated Users:**
   - Automatically redirected from homepage to dashboard
   - Can access all pages including protected dashboard
   - User menu in navbar shows profile with Dashboard link and Sign Out option

## Styling

The application uses modern CSS with:
- Gradient backgrounds
- Smooth transitions and animations
- Responsive design for mobile and desktop
- Clean, minimalist UI inspired by modern SaaS applications

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License
This project is private and proprietary.
