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

// Error handling link
const errorLink = onError(({ graphQLErrors, networkError, operation, forward }) => {
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
      const statusCode = (networkError as any).statusCode;
      if (statusCode === 401 || statusCode === 403) {
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