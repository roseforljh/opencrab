/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Button } from '@douyinfe/semi-ui';

const UsersActions = ({ setShowAddUser, t }) => {
  // Add new user
  const handleAddUser = () => {
    setShowAddUser(true);
  };

  return (
    <div className='flex w-full gap-2 md:w-auto order-2 md:order-1'>
      <Button
        className='!h-11 !w-full !rounded-2xl !border-0 !bg-white/90 hover:!bg-white/80 !px-5 !text-black md:!w-auto'
        onClick={handleAddUser}
        size='small'
      >
        {t('添加用户')}
      </Button>
    </div>
  );
};

export default UsersActions;
