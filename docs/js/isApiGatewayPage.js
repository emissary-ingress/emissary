import ApiGatewayPages from './api-gateway-pages.yml';

const isApiGatewayPage = (path = '') => {
  return ApiGatewayPages.find(p => p.replace(/\//g, '') === path.replace(/\//g, ''));
};

export default isApiGatewayPage;
