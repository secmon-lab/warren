import { ApolloClient, InMemoryCache, createHttpLink } from '@apollo/client';

// 開発時・本番時ともに相対パス'/graphql'を使用してNext.jsのプロキシ機能を活用
const getGraphQLUri = () => {
  return '/graphql';
};

const httpLink = createHttpLink({
  uri: getGraphQLUri(),
});

export const apolloClient = new ApolloClient({
  link: httpLink,
  cache: new InMemoryCache(),
  defaultOptions: {
    watchQuery: {
      errorPolicy: 'ignore',
    },
    query: {
      errorPolicy: 'all',
    },
  },
}); 