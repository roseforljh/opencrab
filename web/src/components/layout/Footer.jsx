
import React, { useEffect, useState } from 'react';
import { getFooterHTML } from '../../helpers';

const FooterBar = () => {
  const [footer, setFooter] = useState(getFooterHTML());

  const loadFooter = () => {
    const footerHtml = localStorage.getItem('footer_html');
    if (footerHtml) {
      setFooter(footerHtml);
    }
  };

  useEffect(() => {
    loadFooter();
  }, []);

  if (!footer) {
    return null;
  }

  return (
    <div className='w-full'>
      <div
        className='custom-footer'
        dangerouslySetInnerHTML={{ __html: footer }}
      ></div>
    </div>
  );
};

export default FooterBar;
