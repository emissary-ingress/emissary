import AesPages from './aes-pages.yml';

const isAesPage = (path = '') => {
  return AesPages.find(p => p.replace(/\//g, '') === path.replace(/\//g, ''));
};

export default isAesPage;
