import { ApolloClient, InMemoryCache, createHttpLink } from '@apollo/client';

// Use relative path '/graphql' for both development and production to leverage Next.js proxy functionality
const getGraphQLUri = () => {
  return '/graphql';
};

const httpLink = createHttpLink({
  uri: getGraphQLUri(),
  credentials: 'include',
});

export const apolloClient = new ApolloClient({
  link: httpLink,
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