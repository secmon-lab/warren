import { ApolloClient, InMemoryCache, createHttpLink, from } from '@apollo/client';
import { onError } from '@apollo/client/link/error';

// Use relative path '/graphql' for both development and production to leverage Next.js proxy functionality
const getGraphQLUri = () => {
  return '/graphql';
};

const httpLink = createHttpLink({
  uri: getGraphQLUri(),
  credentials: 'include',
});

// Prevent infinite redirect loops by tracking redirect attempts
let redirectAttempts = 0;
const MAX_REDIRECT_ATTEMPTS = 1;
const REDIRECT_RESET_TIMEOUT = 30000; // 30 seconds

// Error handling link
const errorLink = onError(({ graphQLErrors, networkError }) => {
  if (graphQLErrors) {
    graphQLErrors.forEach(({ message, locations, path }) => {
      console.error(
        `[GraphQL error]: Message: ${message}, Location: ${locations}, Path: ${path}`
      );
    });
  }

  if (networkError) {
    console.error(`[Network error]: ${networkError}`);
    
    // Handle specific network errors
    if ('statusCode' in networkError) {
      const statusCode = (networkError as { statusCode?: number }).statusCode;
      if (statusCode === 401 || statusCode === 403) {
        // Prevent infinite redirect loops
        if (redirectAttempts >= MAX_REDIRECT_ATTEMPTS) {
          console.error('Too many redirect attempts, avoiding infinite loop');
          return;
        }
        
        // Check if we're already on an auth page to prevent loops
        if (window.location.pathname.startsWith('/api/auth/')) {
          console.error('Already on auth page, avoiding redirect loop');
          return;
        }
        
        redirectAttempts++;
        setTimeout(() => { redirectAttempts = 0; }, REDIRECT_RESET_TIMEOUT);
        
        // Authentication/authorization error - redirect to login
        window.location.href = '/api/auth/login';
        return;
      }
    }
    
    // Handle JSON parse errors from non-JSON responses
    if (networkError.message?.includes('JSON.parse') || 
        networkError.message?.includes('unexpected character')) {
      console.error('Received non-JSON response from GraphQL endpoint. This usually indicates an authentication or server error.');
      
      // Check if we're getting an HTML error page
      if (networkError.message.includes('line 1 column 1')) {
        // Prevent infinite redirect loops
        if (redirectAttempts >= MAX_REDIRECT_ATTEMPTS) {
          console.error('Too many redirect attempts, avoiding infinite loop');
          return;
        }
        
        // Check if we're already on an auth page to prevent loops
        if (window.location.pathname.startsWith('/api/auth/')) {
          console.error('Already on auth page, avoiding redirect loop');
          return;
        }
        
        redirectAttempts++;
        setTimeout(() => { redirectAttempts = 0; }, REDIRECT_RESET_TIMEOUT);
        
        // This is likely an HTML response (like a login page or error page)
        console.error('GraphQL endpoint returned HTML instead of JSON. Redirecting to login...');
        window.location.href = '/api/auth/login';
        return;
      }
    }
  }
});

export const apolloClient = new ApolloClient({
  link: from([errorLink, httpLink]),
  cache: new InMemoryCache(),
  defaultOptions: {
    watchQuery: {
      errorPolicy: 'all',
    },
    query: {
      errorPolicy: 'all',
    },
  },
}); 